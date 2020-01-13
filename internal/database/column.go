package database

import (
	"fmt"
	"math"
	"time"

	"crawshaw.io/sqlite"
)

// GetTime extracts the time from a query result column. NULL becomes
// zero time.
func GetTime(stmt *sqlite.Stmt, param string) (time.Time, error) {
	col := stmt.ColumnIndex(param)
	if col < 0 {
		return time.Time{}, fmt.Errorf("no such column in sql row: %q", param)
	}
	switch columnType := stmt.ColumnType(col); columnType {
	case sqlite.SQLITE_NULL:
		return time.Time{}, nil
	case sqlite.SQLITE_TEXT:
		s := stmt.ColumnText(col)
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("bad time in database: column %s=%q", param, s)
		}
		return t, nil
	default:
		return time.Time{}, fmt.Errorf("bad time in database: column %s type %s value %q", param, columnType, stmt.ColumnText(col))
	}
}

// GetUint8 extracts a uint8 from a query result column, ensuring it
// does not overflow.
func GetUint8(stmt *sqlite.Stmt, param string) (uint8, error) {
	col := stmt.ColumnIndex(param)
	if col < 0 {
		return 0, fmt.Errorf("no such column in sql row: %q", param)
	}
	n := stmt.ColumnInt64(col)
	if n > math.MaxUint8 {
		return 0, fmt.Errorf("bad uint8 in database: column %s=%d", param, n)
	}
	return uint8(n), nil
}
