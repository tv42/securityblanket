package catchup_test

import (
	"context"
	"errors"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"eagain.net/go/securityblanket/internal/catchup"
	"eagain.net/go/securityblanket/internal/database"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
)

func createTable(tb testing.TB, db *database.DB) {
	tb.Helper()
	conn := db.Get(nil)
	defer db.Put(conn)
	if err := sqlitex.ExecScript(conn, `
CREATE TABLE test_source (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	x INTEGER NOT NULL
);
`); err != nil {
		tb.Fatalf("cannot set up schema: %v", err)
	}
}

func makeCatchup(tb testing.TB, db *database.DB) *catchup.Catchup {
	tb.Helper()
	createTable(tb, db)
	c := catchup.New(&catchup.Config{
		DB:     db,
		Log:    zaptest.NewLogger(tb),
		Name:   "xyzzy",
		MaxSQL: `SELECT max(id) AS max FROM test_source`,
		NextSQL: `
SELECT id, x FROM test_source
WHERE id>@last AND id<=@max
ORDER BY id ASC
`,
	})
	return c
}

func execScript(t testing.TB, db *database.DB, sql string) {
	conn := db.Get(nil)
	defer db.Put(conn)
	if err := sqlitex.ExecScript(conn, sql); err != nil {
		t.Fatalf("database error: %v", err)
	}
}

func TestEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	fn := func(conn *sqlite.Conn, stmt *sqlite.Stmt) error {
		t.Fail()
		return errors.New("expected no call on empty database")
	}
	c := makeCatchup(t, db)
	if err := c.Run(ctx, fn); err != nil {
		t.Errorf("catchup run: %v", err)
	}
}

func TestSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	var seen []int64
	fn := func(conn *sqlite.Conn, stmt *sqlite.Stmt) error {
		x := stmt.GetInt64("x")
		seen = append(seen, x)
		return nil
	}
	c := makeCatchup(t, db)
	execScript(t, db, `
INSERT INTO test_source (x) VALUES (10), (11);
`)
	if err := c.Run(ctx, fn); err != nil {
		t.Errorf("catchup run: %v", err)
	}
	want := []int64{10, 11}
	if diff := cmp.Diff(want, seen); diff != "" {
		t.Errorf("wrong results: -want +got\n%s", diff)
	}
}
