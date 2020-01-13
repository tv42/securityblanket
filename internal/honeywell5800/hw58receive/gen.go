package hw58receive

import (
	"crawshaw.io/sqlite"
)

//go:generate go build -o ../../../tools/ github.com/tv42/becky
//go:generate find -name "[a-zA-Z]*.sql" -exec ../../../tools/becky -wrap=sqlAsset {} +

type sqlAsset asset

func (a sqlAsset) Prep(conn *sqlite.Conn) *sqlite.Stmt {
	// TODO load from disk in dev mode, without go generate; maybe add
	// String methods to dev/nodev asset, take fmt.Stringer here
	return conn.Prep(a.Content)
}
