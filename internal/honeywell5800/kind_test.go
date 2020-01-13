package honeywell5800_test

import (
	"testing"

	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800"
)

func TestKindKnownFromDB(t *testing.T) {
	db := database.Scratch()
	defer db.Close()
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT id AS kind FROM honeywell5800_loop_kinds
`)
	defer stmt.Finalize()
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			t.Fatalf("database error: %v", err)
		}
		if !hasRow {
			break
		}
		k, err := honeywell5800.KindFromSQL(stmt, "kind")
		if err != nil {
			t.Errorf("did not recognize kind: %q", stmt.GetText("kind"))
		}
		if g, e := k.String(), stmt.GetText("kind"); g != e {
			t.Errorf("kind did not roundtrip: %q != %q", g, e)
		}
	}
}

func TestKindKnownTowardDB(t *testing.T) {
	db := database.Scratch()
	defer db.Close()
	conn := db.Get(nil)
	defer db.Put(conn)
	stmt := conn.Prep(`
SELECT 1 FROM honeywell5800_loop_kinds WHERE id=@kind
`)
	defer stmt.Finalize()
	for _, kind := range honeywell5800.KindValues() {
		if err := stmt.Reset(); err != nil {
			t.Fatalf("database error: %v", err)
		}
		stmt.SetText("@kind", kind.String())
		for {
			hasRow, err := stmt.Step()
			if err != nil {
				t.Fatalf("database error: %v", err)
			}
			if !hasRow {
				t.Errorf("kind not in database: %q", kind)
				break
			}
			// success, 1 row is enough
			break
		}
	}
}
