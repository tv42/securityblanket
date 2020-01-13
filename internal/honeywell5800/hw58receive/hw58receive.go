package hw58receive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"crawshaw.io/sqlite"
	"eagain.net/go/securityblanket/internal/catchup"
	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800"
	"eagain.net/go/securityblanket/internal/jsonx"
	"go.uber.org/zap"
)

var (
	errDuplicate = errors.New("duplicate sensor update")
)

type Receiver struct {
	ctx     context.Context
	catchup *catchup.Catchup
	log     *zap.Logger
	wakeup  func()
}

func New(ctx context.Context, db *database.DB, log *zap.Logger, wakeup func()) *Receiver {
	r := &Receiver{
		ctx: ctx,
		catchup: catchup.New(&catchup.Config{
			DB:      db,
			Log:     log.Named("catchup"),
			Name:    "honeywell5800.receive",
			MaxSQL:  fetch_rtl433_raw_max.Content,
			NextSQL: fetch_rtl433_raw.Content,
		}),
		log:    log,
		wakeup: wakeup,
	}
	return r
}

type rtl433Message struct {
	ID      honeywell5800.Sensor
	Channel honeywell5800.Channel
	Event   honeywell5800.Event
}

func parseRTL433(data []byte) (*rtl433Message, error) {
	var msg rtl433Message
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&msg); err != nil {
		return nil, fmt.Errorf("cannot parse rtl_433 output: %w", err)
	}
	if err := jsonx.MustEOF(dec); err != nil {
		return nil, fmt.Errorf("trailing junk in rtl_433 output line: %w", err)
	}
	return &msg, nil
}

// add the sensor to the database, if it doesn't exist already
func insertSensor(conn *sqlite.Conn, sensor honeywell5800.Sensor, created time.Time) error {
	stmt := insert_honeywell5800_sensor.Prep(conn)
	defer stmt.Finalize()
	sensor.ToSQL(stmt, "@sensor")
	database.BindTime(stmt, "@created", created)
	if _, err := stmt.Step(); err != nil {
		return err
	}
	return nil
}

// addUpdate adds a sensor update to the database.
//
// Returns errDuplicate if the update has been seen already, as is
// very common with rapidly repeated one-way radio transmissions. Most
// callers should quietly stop further processing.
func addUpdate(ctx context.Context, conn *sqlite.Conn, ts time.Time, update *rtl433Message) error {
	// no sqlite savepoint because any progress is useful and
	// idempotent

	if err := insertSensor(conn, update.ID, ts); err != nil {
		return fmt.Errorf("adding sensor: %w", err)
	}

	const dedupWindow = 5 * time.Second
	stmt := insert_honeywell5800_update.Prep(conn)
	defer stmt.Finalize()
	database.BindTime(stmt, "@time", ts)
	database.BindTime(stmt, "@dedupTime", ts.Add(-dedupWindow))
	update.Channel.ToSQL(stmt, "@channel")
	update.ID.ToSQL(stmt, "@sensor")
	update.Event.ToSQL(stmt, "@event")
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("add sensor update: %w", err)
	}
	switch affected := conn.Changes(); affected {
	case 0:
		// deduplicated
		return errDuplicate
	case 1:
		return nil
	default:
		return fmt.Errorf("internal error: sensor dedup caused multiple rows: %d", affected)
	}
}

func (r *Receiver) Run() error {
	return r.catchup.Run(r.ctx, r.run)
}

func (r *Receiver) run(conn *sqlite.Conn, stmt *sqlite.Stmt) error {
	r.log.Debug("working")
	defer r.log.Debug("done")
	ts, err := database.GetTime(stmt, "time")
	if err != nil {
		return fmt.Errorf("parsing rtl433 update: %w", err)
	}
	data, _ := ioutil.ReadAll(stmt.GetReader("data"))

	update, err := parseRTL433(data)
	if err != nil {
		return fmt.Errorf("error parsing rtl433 honeywell5800 message: %w", err)
	}
	r.log.Debug("received",
		zap.Stringer("sensor", update.ID),
		zap.Uint8("channel", uint8(update.Channel)),
		zap.Stringer("event", update.Event),
		zap.String("event.parsed", fmt.Sprintf("%+v", update.Event)),
	)
	if err := addUpdate(r.ctx, conn, ts, update); err != nil {
		if errors.Is(err, errDuplicate) {
			return nil
		}
		return fmt.Errorf("error adding sensor update: %w", err)
	}

	r.wakeup()
	return nil
}
