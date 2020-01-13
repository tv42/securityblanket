package rtl433sql

import (
	"context"
	"fmt"
	"time"

	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/rtl433receive"
)

type config struct {
	wakeup func()
	clock  func() time.Time
}

type SQLStore struct {
	db      *database.DB
	freqMHz int64
	config  config
}

type Option option

type option func(*config)

func Clock(clock func() time.Time) Option {
	fn := func(conf *config) {
		conf.clock = clock
	}
	return fn
}

func Wakeup(wakeup func()) Option {
	fn := func(conf *config) {
		conf.wakeup = wakeup
	}
	return fn
}

func New(db *database.DB, freqMHz int64, opts ...Option) *SQLStore {
	h := &SQLStore{
		db:      db,
		freqMHz: freqMHz,
		config: config{
			wakeup: func() {},
			clock:  time.Now,
		},
	}
	for _, opt := range opts {
		opt(&h.config)
	}
	return h
}

var _ rtl433receive.Store = (*SQLStore)(nil)

func (s *SQLStore) Store(ctx context.Context, data []byte) error {
	now := s.config.clock()

	conn := s.db.Get(ctx)
	if conn == nil {
		return context.Canceled
	}
	defer s.db.Put(conn)

	stmt := insert_rtl433_raw.Prep(conn)
	defer stmt.Finalize()
	stmt.SetText("@time", now.Format(time.RFC3339Nano))
	stmt.SetInt64("@freqMHz", s.freqMHz)
	stmt.SetBytes("@data", data)
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("cannot insert rtl_433 345MHz raw data: %w", err)
	}

	switch affected := conn.Changes(); affected {
	case 0:
		// deduplicated; do nothing
	case 1:
		s.config.wakeup()
	default:
		return fmt.Errorf("internal error: rtl433 dedup caused multiple rows: %d", affected)
	}
	return nil
}
