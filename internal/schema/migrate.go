package schema

import (
	"fmt"
	"strconv"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

//go:generate go build -o ../../tools/ github.com/tv42/becky
//go:generate find -name "[a-zA-Z]*.sql" -exec ../../tools/becky -wrap=sqlAsset {} +
//go:generate find -name "[0-9]*.sql" -exec ../../tools/becky -var=_ -wrap=addMigration {} +

func sqlAsset(a asset) string {
	return a.Content
}

var migrations []string

func addMigration(a asset) struct{} {
	name := strings.TrimSuffix(a.Name, ".sql")
	num, err := strconv.ParseUint(name, 10, strconv.IntSize)
	if err != nil {
		panic(fmt.Errorf("migration filename is not a number: %q: %v", a.Name, err))
	}
	if num == 0 {
		panic(fmt.Errorf("number migrations starting from 1: %q", a.Name))
	}
	// cannot overflow because of strconv bitsize restriction
	n := int(num)
	if len(migrations) <= n {
		need := n - len(migrations) + 1
		migrations = append(migrations, make([]string, need)...)
	}
	migrations[n] = a.Content
	return struct{}{}
}

func Migrate(conn *sqlite.Conn) error {
	if err := sqlitex.ExecScript(conn, create_schema_version); err != nil {
		return fmt.Errorf("create schema migration state: %v", err)
	}

	stmt := conn.Prep(get_schema_version)
	defer stmt.Finalize()
	version, err := sqlitex.ResultInt64(stmt)
	if err != nil {
		return fmt.Errorf("cannot fetch current schema version: %v", err)
	}
	if version < 0 {
		return fmt.Errorf("schema version cannot be negative: %d", version)
	}
	if version > int64(len(migrations)) {
		return fmt.Errorf("schema version is greater than what we know: %d", version)
	}

	for i := version + 1; i < int64(len(migrations)); i++ {
		m := migrations[i]
		if m == "" {
			return fmt.Errorf("migration is empty: #%d", i)
		}
		if err := migrateStep(conn, i, m); err != nil {
			return fmt.Errorf("step failed: #%d: %v\n%s", i, err, m)
		}
	}

	return nil
}

func migrateStep(conn *sqlite.Conn, version int64, query string) (err error) {
	defer sqlitex.Save(conn)(&err)

	if err := sqlitex.ExecScript(conn, query); err != nil {
		return fmt.Errorf("create schema migration state: %w", err)
	}

	stmt := conn.Prep(update_schema_version)
	defer stmt.Finalize()
	stmt.SetInt64("@newVersion", version)
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("updating schema version: %w", err)
	}

	return nil
}
