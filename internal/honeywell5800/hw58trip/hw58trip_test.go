package hw58trip_test

import (
	"context"
	"testing"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800/hw58trip"
	"go.uber.org/zap/zaptest"
)

func count(t testing.TB, db *database.DB) int64 {
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT count(*) AS count FROM honeywell5800_trips
`)
	defer stmt.Finalize()
	if err := database.Row(stmt); err != nil {
		t.Fatalf("database error when counting: %v", err)
	}
	n := stmt.GetInt64("count")
	if err := database.NoMoreRows(stmt); err != nil {
		t.Fatalf("database error: %v", err)
	}
	return n
}

func execScript(t testing.TB, db *database.DB, sql string) {
	conn := db.Get(nil)
	defer db.Put(conn)

	if err := sqlitex.ExecScript(conn, sql); err != nil {
		t.Fatalf("database error: %v", err)
	}
}

func TestTrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	defer db.Close()

	now := time.Date(2020, 2, 3, 4, 5, 6, 7, time.Local)
	log := zaptest.NewLogger(t)
	trip := hw58trip.New(ctx, db, log)

	// nothing to do
	if err := trip.Run(); err != nil {
		t.Fatalf("run empty: %v", err)
	}
	if g, e := count(t, db), int64(0); g != e {
		t.Errorf("wrong number of results: %d != %d", g, e)
	}

	execScript(t, db, `
INSERT INTO honeywell5800_sensors(id, model, description)
VALUES (123456,'5853','west wing');

INSERT INTO honeywell5800_updates(id, time, channel, sensor, event)
VALUES (42, '`+now.Format(time.RFC3339)+`', 8, 123456, 128);
`)
	if err := trip.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if g, e := count(t, db), int64(1); g != e {
		t.Errorf("wrong number of results: %d != %d", g, e)
	}
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT * FROM honeywell5800_trips
`)
	defer stmt.Finalize()
	if err := database.Row(stmt); err != nil {
		t.Fatalf("database error reading updates: %v", err)
	}
	if g, e := stmt.ColumnCount(), 5; g != e {
		t.Errorf("wrong number of columns: %d != %d", g, e)
	}
	// don't care about id
	if g, e := stmt.GetInt64("sensor"), int64(123456); g != e {
		t.Errorf("wrong sensor: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("loop"), int64(1); g != e {
		t.Errorf("wrong loop: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("trippedBy"), int64(42); g != e {
		t.Errorf("wrong trippedBy: %v != %v", g, e)
	}
	if stmt.ColumnType(stmt.ColumnIndex("clearedBy")) != sqlite.SQLITE_NULL {
		t.Errorf("not NULL clearedBy: %v", stmt.GetText("clearedBy"))
	}
	if err := database.NoMoreRows(stmt); err != nil {
		t.Fatalf("database error: %v", err)
	}
}

func TestNormal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	defer db.Close()

	now := time.Date(2020, 2, 3, 4, 5, 6, 7, time.Local)
	log := zaptest.NewLogger(t)
	trip := hw58trip.New(ctx, db, log)

	execScript(t, db, `
INSERT INTO honeywell5800_sensors(id, model, description)
VALUES (123456,'5853','west wing');

INSERT INTO honeywell5800_updates(id, time, channel, sensor, event)
VALUES (42, '`+now.Format(time.RFC3339)+`', 8, 123456, 128);

INSERT INTO honeywell5800_updates(id, time, channel, sensor, event)
VALUES (43, '`+now.Format(time.RFC3339)+`', 8, 123456, 0);

INSERT INTO honeywell5800_trips(sensor, loop, trippedBy)
VALUES (123456, 1, 42);

INSERT INTO catchup (name, last) VALUES ('honeywell5800.trip', 42);
`)
	if err := trip.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if g, e := count(t, db), int64(1); g != e {
		t.Errorf("wrong number of results: %d != %d", g, e)
	}
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT * FROM honeywell5800_trips
`)
	defer stmt.Finalize()
	if err := database.Row(stmt); err != nil {
		t.Fatalf("database error reading updates: %v", err)
	}
	if g, e := stmt.ColumnCount(), 5; g != e {
		t.Errorf("wrong number of columns: %d != %d", g, e)
	}
	// don't care about id
	if g, e := stmt.GetInt64("sensor"), int64(123456); g != e {
		t.Errorf("wrong sensor: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("loop"), int64(1); g != e {
		t.Errorf("wrong loop: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("trippedBy"), int64(42); g != e {
		t.Errorf("wrong trippedBy: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("clearedBy"), int64(43); g != e {
		t.Errorf("wrong clearedBy: %v != %v", g, e)
	}
	if err := database.NoMoreRows(stmt); err != nil {
		t.Fatalf("database error: %v", err)
	}
}
