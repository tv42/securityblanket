package honeywell5800

import (
	"fmt"

	"crawshaw.io/sqlite"
)

//go:generate go run github.com/alvaroloes/enumer -type=Kind -output=kind.gen.go -linecomment

// Kind stores the kind of a sensor loop.
type Kind int

const (
	_                 Kind = iota
	Door                   // door open
	DoorWindow             // door or window open
	GlassBreak             // glass break
	HeatDetector           // heat detector
	KeyFob                 // key fob button
	LowTemp                // low temperature
	MaintenanceNeeded      // maintenance needed
	MedicalAlert           // medical alert
	MotionDetector         // motion detector
	PanicButton            // panic button
	SmokeDetector          // smoke detector
	Tamper                 // tamper
	TiltSwitch             // tilt switch
	Window                 // window open
)

func KindFromSQL(stmt *sqlite.Stmt, param string) (Kind, error) {
	col := stmt.ColumnIndex(param)
	if col < 0 {
		return 0, fmt.Errorf("no such column in sql row: %q", param)
	}
	s := stmt.ColumnText(col)
	k, err := KindString(s)
	if err != nil {
		return Kind(0), fmt.Errorf("bad sensor kind in database: column %s=%q", param, s)
	}
	return k, nil
}
