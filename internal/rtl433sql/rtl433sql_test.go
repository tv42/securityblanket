package rtl433sql_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/rtl433sql"
)

func TestSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := database.Scratch()
	defer db.Close()

	var wakeups uint64
	wakeup := func() {
		atomic.AddUint64(&wakeups, 1)
	}
	const freq = 123
	now := time.Date(2020, 1, 2, 3, 4, 5, 6, time.Local)
	clock := func() time.Time { return now }
	s := rtl433sql.New(db, freq,
		rtl433sql.Wakeup(wakeup),
		rtl433sql.Clock(clock),
	)
	const input = `{"model": "xyzzy", "foo": 42}`
	if err := s.Store(ctx, []byte(input)); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if g, e := atomic.LoadUint64(&wakeups), uint64(1); g != e {
		t.Errorf("wrong number of wakeups: %d != %d", g, e)
	}
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`SELECT * FROM rtl433_raw`)
	defer stmt.Finalize()
	hasRow, err := stmt.Step()
	if err != nil {
		t.Fatalf("database error: %v", err)
	}
	if !hasRow {
		t.Fatal("expected 1 row, got none")
	}
	if g, e := stmt.ColumnCount(), 5; g != e {
		t.Errorf("wrong number of columns: %d != %d", g, e)
	}
	// don't care about id
	if g, e := stmt.GetText("time"), now.Format(time.RFC3339Nano); g != e {
		t.Errorf("wrong time: %v != %v", g, e)
	}
	if g, e := stmt.GetInt64("freqMHz"), int64(freq); g != e {
		t.Errorf("wrong freqMHz: %v != %v", g, e)
	}
	if g, e := stmt.GetText("model"), "xyzzy"; g != e {
		t.Errorf("wrong model: %v != %v", g, e)
	}
	if g, e := stmt.GetText("data"), `{"foo":42}`; g != e {
		t.Errorf("wrong data: %v != %v", g, e)
	}
}
