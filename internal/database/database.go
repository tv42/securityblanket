package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync/atomic"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"eagain.net/go/securityblanket/internal/schema"
)

func makeURL(dbPath string) url.URL {
	q := make(url.Values)
	// Not at all convinced this does anything. I can't find this in
	// the official SQLite docs, but I do find it in lots of rumors
	// around the internet.
	q.Set("timezone", "auto")
	u := url.URL{
		Scheme: "file",
		// Use Opaque not Path or net/url will make it file://foo
		// which would prevent using relative filenames
		Opaque:   dbPath,
		RawQuery: q.Encode(),
	}
	return u
}

type DB struct {
	*sqlitex.Pool
}

func (db DB) Get(ctx context.Context) *sqlite.Conn {
	conn := db.Pool.Get(ctx)
	if conn == nil {
		return nil
	}

	// technically this is done too often, but sqlitex.Pool does not
	// expose anything that would let us differentiate between new and
	// old connections.
	//
	// i have been tempted to avoid sqlitex.Pool as a whole, but it
	// seems to contain a lot of careful code.
	stmt, _, err := conn.PrepareTransient("PRAGMA foreign_keys=1;")
	if err != nil {
		panic(err)
	}
	defer stmt.Finalize()
	if _, err := stmt.Step(); err != nil {
		panic(err)
	}

	return conn
}

func openDB(u url.URL, flags sqlite.OpenFlags) (*DB, error) {
	const baseFlags = 0 |
		sqlite.SQLITE_OPEN_READWRITE |
		sqlite.SQLITE_OPEN_CREATE |
		sqlite.SQLITE_OPEN_WAL |
		sqlite.SQLITE_OPEN_URI |
		sqlite.SQLITE_OPEN_NOMUTEX |
		sqlite.SQLITE_OPEN_SHAREDCACHE |
		0
	const poolSize = 10
	pool, err := sqlitex.Open(u.String(), baseFlags|flags, poolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %v", err)
	}
	success := false
	defer func() {
		if !success {
			_ = pool.Close()
		}
	}()

	conn := pool.Get(nil)
	defer pool.Put(conn)
	if err := schema.Migrate(conn); err != nil {
		return nil, fmt.Errorf("cannot migrate sql schema: %v", err)
	}

	success = true
	db := &DB{Pool: pool}
	return db, nil
}

func Open(dbPath string) (*DB, error) {
	u := makeURL(dbPath)
	return openDB(u, 0)
}

// separate in-memory databases for every Scratch call
var inmemCounter uint64

func Scratch() *DB {
	n := atomic.AddUint64(&inmemCounter, 1)
	p := strconv.FormatUint(n, 10)
	u := makeURL(p)
	const flags = 0 |
		sqlite.SQLITE_OPEN_SHAREDCACHE |
		sqlite.SQLITE_OPEN_MEMORY |
		0
	db, err := openDB(u, flags)
	if err != nil {
		panic(fmt.Errorf("OpenDBInMemory: %w", err))
	}
	return db
}

// Row fetches the next row in the query results. It returns an error
// if there is no next row.
func Row(stmt *sqlite.Stmt) error {
	hasRow, err := stmt.Step()
	if err != nil {
		return err
	}
	if !hasRow {
		return errors.New("no row found in database")
	}
	return nil
}

// NoMoreRows returns an error if there are still more rows in the
// query results.
func NoMoreRows(stmt *sqlite.Stmt) error {
	hasRow, err := stmt.Step()
	if err != nil {
		return err
	}
	if hasRow {
		return errors.New("too many rows found in database")
	}
	return nil
}
