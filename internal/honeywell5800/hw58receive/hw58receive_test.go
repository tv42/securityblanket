package hw58receive_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800/hw58receive"
	"eagain.net/go/securityblanket/internal/rtl433sql"
	"go.uber.org/zap/zaptest"
)

func count(t testing.TB, db *database.DB) int64 {
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT count(*) AS count FROM honeywell5800_updates
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

func TestSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	defer db.Close()

	now := time.Date(2020, 2, 3, 4, 5, 6, 7, time.Local)
	clock := func() time.Time { return now }
	log := zaptest.NewLogger(t)
	var wakeups uint64
	wakeup := func() {
		atomic.AddUint64(&wakeups, 1)
	}
	recv := hw58receive.New(ctx, db, log, wakeup)
	store := rtl433sql.New(db, 123, rtl433sql.Clock(clock))
	if err := store.Store(ctx, []byte(`{"model": "junk"}`)); err != nil {
		t.Fatalf("store: %v", err)
	}

	// nothing to do
	if err := recv.Run(); err != nil {
		t.Fatalf("run empty: %v", err)
	}
	if g, e := count(t, db), int64(0); g != e {
		t.Errorf("wrong number of results: %d != %d", g, e)
	}
	if g, e := atomic.LoadUint64(&wakeups), uint64(0); g != e {
		t.Errorf("wrong number of wakeups: %d != %d", g, e)
	}

	if err := store.Store(ctx, []byte(`{"model": "Honeywell-Security", "channel": 3, "id": 123456, "event": 128}`)); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := recv.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if g, e := count(t, db), int64(1); g != e {
		t.Errorf("wrong number of results: %d != %d", g, e)
	}
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT * FROM honeywell5800_updates
`)
	defer stmt.Finalize()
	if err := database.Row(stmt); err != nil {
		t.Fatalf("database error reading updates: %v", err)
	}
	if g, e := stmt.ColumnCount(), 5; g != e {
		t.Errorf("wrong number of columns: %d != %d", g, e)
	}
	// don't care about id
	if g, e := stmt.GetText("time"), now.Format(time.RFC3339Nano); g != e {
		t.Errorf("wrong time: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("channel"), int64(3); g != e {
		t.Errorf("wrong channel: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("sensor"), int64(123456); g != e {
		t.Errorf("wrong sensor: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("event"), int64(0x80); g != e {
		t.Errorf("wrong event: %v != %v", g, e)
	}
	if err := database.NoMoreRows(stmt); err != nil {
		t.Fatalf("database error: %v", err)
	}
	if g, e := atomic.LoadUint64(&wakeups), uint64(1); g != e {
		t.Errorf("wrong number of wakeups: %d != %d", g, e)
	}
}
