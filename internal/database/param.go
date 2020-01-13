package database

import (
	"time"

	"crawshaw.io/sqlite"
)

// BindTime sets the time as a SQL query bind parameter.
// Zero time sets NULL.
func BindTime(stmt *sqlite.Stmt, param string, t time.Time) {
	if t.IsZero() {
		stmt.SetNull(param)
		return
	}
	stmt.SetText(param, t.Format(time.RFC3339Nano))
}
