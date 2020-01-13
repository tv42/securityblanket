// Package catchup implements log-oriented data processing on top of
// SQLite. Rows from an append-only table are processed incrementally.
//
// Source table MUST be append-only, with the following exceptions.
//
// 1. Rows CAN be deleted, but consumers will NOT be notified, and
// those rows may or may not have been seen. Deleting rows with id <=
// minimum of last seen of all consumers SHOULD be done to prevent
// unlimited database growth.
//
// 2. Rows CAN be updated, but consumers will NOT be notified; hence
// only columns not directly used by consumers are good candidates.
//
// Source table MUST NOT be inserted into concurrently (this can cause
// ids to show up out of order).
//
// Source tables that see any deletes (including pruning of oldest
// entries) MUST use AUTOINCREMENT.
//
// Destination table and catchup's internal bookkeeping MUST be
// protected by the same SQLite savepoint (which roughly means they
// must be in the same database, and changed through the same database
// connection).
package catchup

import (
	"context"
	"errors"
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"eagain.net/go/securityblanket/internal/database"
	"go.uber.org/zap"
)

type Config struct {
	DB   *database.DB
	Log  *zap.Logger
	Name string
	// MaxSQL is the SQL query to fetch the maximum ID in a source
	// table. The result must have column named max.
	MaxSQL string
	// NextSQL is the SQL query to fetch rows from the source table.
	// The result must have column named id, and should use bind
	// parameters @last and @max to limit the rows.
	NextSQL string
}

type Catchup struct {
	conf Config
}

func New(conf *Config) *Catchup {
	c := &Catchup{
		conf: *conf,
	}
	return c
}

func fetchMax(conn *sqlite.Conn, sql string) (int64, error) {
	stmt := conn.Prep(sql)
	defer stmt.Finalize()
	hasRow, err := stmt.Step()
	if err != nil {
		return 0, err
	}
	if !hasRow {
		// no rows at all -> bail out
		return 0, nil
	}
	col := stmt.ColumnIndex("max")
	if col < 0 {
		return 0, errors.New("max id SQL query must return a column named max")
	}
	max := stmt.ColumnInt64(col)
	if err := database.NoMoreRows(stmt); err != nil {
		return 0, err
	}
	return max, nil
}

func (c *Catchup) load(conn *sqlite.Conn) (int64, error) {
	stmt := load.Prep(conn)
	defer stmt.Finalize()
	stmt.SetText("@name", c.conf.Name)
	if err := database.Row(stmt); err != nil {
		return 0, err
	}
	last := stmt.GetInt64("last")
	if err := database.NoMoreRows(stmt); err != nil {
		return 0, fmt.Errorf("more than one row: %w", err)
	}
	return last, nil
}

func (c *Catchup) save(conn *sqlite.Conn, last int64) error {
	stmt := save.Prep(conn)
	defer stmt.Finalize()
	stmt.SetText("@name", c.conf.Name)
	stmt.SetInt64("@last", last)
	if _, err := stmt.Step(); err != nil {
		return err
	}
	return nil
}

func (c *Catchup) runRow(conn *sqlite.Conn, fn Func, stmt *sqlite.Stmt, id int64) (err error) {
	defer sqlitex.Save(conn)(&err)
	if err := fn(conn, stmt); err != nil {
		return fmt.Errorf("error from user function: %w", err)
	}
	if err := c.save(conn, id); err != nil {
		return fmt.Errorf("saving last processed id: %w", err)
	}
	return nil
}

func (c *Catchup) run(ctx context.Context, fn Func) (progress bool, err error) {
	conn := c.conf.DB.Get(ctx)
	if conn == nil {
		return false, context.Canceled
	}
	defer c.conf.DB.Put(conn)

	max, err := fetchMax(conn, c.conf.MaxSQL)
	if err != nil {
		return false, fmt.Errorf("fetching max id: %w", err)
	}

	last, err := c.load(conn)
	if err != nil {
		return false, fmt.Errorf("fetching last processed id: %w", err)
	}
	madeProgress := false
	stmt := conn.Prep(c.conf.NextSQL)
	defer stmt.Finalize()
	stmt.SetInt64("@last", last)
	stmt.SetInt64("@max", max)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return madeProgress, err
		}
		if !hasRow {
			break
		}
		id := stmt.GetInt64("id")
		if err := c.runRow(conn, fn, stmt, id); err != nil {
			return madeProgress, err
		}
		last = id
		madeProgress = true
	}
	return madeProgress, nil
}

// Func is a function that does the actual work. The current source
// row is accessible via stmt, and all database updates should be done
// through conn.
//
// The function may be called multiple times for the same input. All
// side effects must happen either through conn, or be idempotent.
type Func func(conn *sqlite.Conn, stmt *sqlite.Stmt) error

func (c *Catchup) Run(ctx context.Context, fn Func) (err error) {
	for {
		progress, err := c.run(ctx, fn)
		if err != nil {
			var sqerr sqlite.Error
			if errors.As(err, &sqerr) && sqerr.Code == sqlite.SQLITE_LOCKED {
				c.conf.Log.Debug("retry.sqlite_deadlock",
					zap.Stringer("code", sqerr.Code),
					zap.String("msg", sqerr.Msg),
					zap.String("loc", sqerr.Loc),
					zap.String("query", sqerr.Query),
				)
				continue
			}
			return fmt.Errorf("catchup %v: %w", c.conf.Name, err)
		}
		if !progress {
			break
		}
	}
	return nil
}
