package hw58trip

import (
	"context"
	"errors"
	"fmt"

	"crawshaw.io/sqlite"
	"eagain.net/go/securityblanket/internal/catchup"
	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800"
	"go.uber.org/zap"
)

var (
	errDuplicate = errors.New("duplicate sensor update")
)

type Tripper struct {
	ctx     context.Context
	catchup *catchup.Catchup
	log     *zap.Logger
}

func New(ctx context.Context, db *database.DB, log *zap.Logger) *Tripper {
	t := &Tripper{
		ctx: ctx,
		catchup: catchup.New(&catchup.Config{
			DB:      db,
			Log:     log.Named("catchup"),
			Name:    "honeywell5800.trip",
			MaxSQL:  fetch_honeywell5800_updates_max.Content,
			NextSQL: fetch_honeywell5800_updates.Content,
		}),
		log: log,
	}
	return t
}

func (t *Tripper) Run() error {
	return t.catchup.Run(t.ctx, t.run)
}

func (t *Tripper) run(conn *sqlite.Conn, stmt *sqlite.Stmt) error {
	t.log.Info("working")
	defer t.log.Info("done")
	updateID := stmt.GetInt64("id")
	sensor := honeywell5800.SensorFromSQL(stmt, "sensor")
	event := honeywell5800.EventFromSQL(stmt, "event")

	loopStmt := fetch_honeywell5800_loops.Prep(conn)
	defer loopStmt.Finalize()
	sensor.ToSQL(loopStmt, "@sensor")
	for {
		hasRow, err := loopStmt.Step()
		if err != nil {
			return fmt.Errorf("error fetching sensor loops: %v: %w", sensor, err)
		}
		if !hasRow {
			break
		}

		model := loopStmt.GetText("model")
		description := loopStmt.GetText("description")
		loop, err := database.GetUint8(loopStmt, "loop")
		if err != nil {
			return fmt.Errorf("bad loop in database: sensor %v: %w", sensor, err)
		}
		kind, err := honeywell5800.KindFromSQL(loopStmt, "kind")
		if err != nil {
			return fmt.Errorf("bad kind in database: sensor %v loop %d: %w", sensor, loop, err)
		}
		label := loopStmt.GetText("label")
		normallyOpen := loopStmt.GetInt64("normallyOpen") != 0

		isOpen := event.Loop(loop)
		isTrip := isOpen != normallyOpen

		// log per-loop update here, then again in Trip/Normal when
		// the state actuall changes
		t.log.Debug("update",
			zap.Bool("tripped", isTrip),
			zap.Stringer("sensor", sensor),
			zap.String("model", model),
			zap.String("description", description),
			zap.Uint8("loop", loop),
			zap.Stringer("kind", kind),
			zap.String("label", label),
		)

		switch {
		case isTrip:
			err := t.trip(conn,
				sensor, model, description,
				loop, kind, label,
				updateID)
			switch err {
			case nil:
				t.log.Info("trip",
					zap.Stringer("sensor", sensor),
					zap.String("model", model),
					zap.String("description", description),
					zap.Uint8("loop", loop),
					zap.Stringer("kind", kind),
					zap.String("label", label),
				)
				// wakeup anyone after us in the pipeline; nobody
				// there yet
			case errDuplicate:
				// nothing
			default:
				return err
			}

		case !isTrip:
			// the trip is cleared
			err := t.normal(conn,
				sensor, model, description,
				loop, kind, label,
				updateID)
			switch err {
			case nil:
				// log more information about the original Trip? duration
				// tripped etc.
				t.log.Info("normal",
					zap.Stringer("sensor", sensor),
					zap.String("model", model),
					zap.String("description", description),
					zap.Uint8("loop", loop),
					zap.Stringer("kind", kind),
					zap.String("label", label),
				)
				// wakeup anyone after us in the pipeline; nobody
				// there yet
			case errDuplicate:
			// nothing
			default:
				return err
			}
		}
	}
	return nil
}

func (t *Tripper) trip(
	conn *sqlite.Conn,
	sensor honeywell5800.Sensor,
	model string,
	description string,
	loop uint8,
	kind honeywell5800.Kind,
	label string,
	updateID int64,
) error {
	stmt := insert_honeywell5800_trip.Prep(conn)
	defer stmt.Finalize()
	sensor.ToSQL(stmt, "@sensor")
	stmt.SetInt64("@loop", int64(loop))
	stmt.SetInt64("@trippedBy", updateID)
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("add trip: %w", err)
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

func (t *Tripper) normal(
	conn *sqlite.Conn,
	sensor honeywell5800.Sensor,
	model string,
	description string,
	loop uint8,
	kind honeywell5800.Kind,
	label string,
	updateID int64,
) error {
	stmt := update_honeywell5800_trip_cleared.Prep(conn)
	defer stmt.Finalize()
	sensor.ToSQL(stmt, "@sensor")
	stmt.SetInt64("@loop", int64(loop))
	stmt.SetInt64("@clearedBy", updateID)
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("clear trip: %w", err)
	}
	switch affected := conn.Changes(); affected {
	case 0:
		// there was no trip to clear
		//
		// strictly, duplicate is not the only reason for that to
		// happen, but it's fine.
		return errDuplicate
	case 1:
		return nil
	default:
		return fmt.Errorf("internal error: clearing trip caused multiple changes: %d", affected)
	}
}
