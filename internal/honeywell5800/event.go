package honeywell5800

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"crawshaw.io/sqlite"
)

type Event uint8

const (
	// event bits that seem common to all devices
	eventRegister   Event = 0x02
	eventHeartbeat  Event = 0x04
	eventBatteryLow Event = 0x08

	// the meaning of the 4 "loops" is device-dependent.
	eventLoop1 Event = 0x80
	eventLoop2 Event = 0x20
	eventLoop3 Event = 0x10
	eventLoop4 Event = 0x40

	eventMask Event = 0 |
		eventRegister |
		eventHeartbeat |
		eventBatteryLow |
		eventLoop1 |
		eventLoop2 |
		eventLoop3 |
		eventLoop4 |
		0
)

func (e Event) IsRegister() bool {
	return e&eventRegister != 0
}

func (e Event) IsBatteryLow() bool {
	return e&eventBatteryLow != 0
}

func (e Event) IsHeartbeat() bool {
	return e&eventHeartbeat != 0
}

func (e Event) Loop(n uint8) bool {
	switch n {
	default:
		return false
	case 1:
		return e.Loop1()
	case 2:
		return e.Loop2()
	case 3:
		return e.Loop3()
	case 4:
		return e.Loop4()
	}
}

func (e Event) Loop1() bool {
	return e&eventLoop1 != 0
}

func (e Event) Loop2() bool {
	return e&eventLoop2 != 0
}

func (e Event) Loop3() bool {
	return e&eventLoop3 != 0
}

func (e Event) Loop4() bool {
	return e&eventLoop4 != 0
}

var _ fmt.Stringer = Event(0)

func (e Event) String() string {
	return fmt.Sprintf("%#02x", uint8(e))
}

func (e Event) dump() string {
	if e == 0 {
		return "none"
	}
	var b strings.Builder
	if e.Loop1() {
		b.WriteString("+L1")
	}
	if e.Loop2() {
		b.WriteString("+L2")
	}
	if e.Loop3() {
		b.WriteString("+L3")
	}
	if e.Loop4() {
		b.WriteString("+L4")
	}
	if e.IsRegister() {
		b.WriteString("+register")
	}
	if e.IsBatteryLow() {
		b.WriteString("+battery")
	}
	if e.IsHeartbeat() {
		b.WriteString("+heartbeat")
	}
	if u := e & ^eventMask; u != 0 {
		fmt.Fprintf(&b, "+%#02x", uint32(u))
	}
	return b.String()[1:]
}

var _ fmt.Formatter = Event(0)

func (e Event) Format(state fmt.State, verb rune) {
	if verb != 'v' {
		panic("honeywell5800.Event.Format only handles %v")
	}
	if !state.Flag('+') {
		_, _ = io.WriteString(state, e.String())
		return
	}
	_, _ = io.WriteString(state, e.dump())
}

func (e Event) ToSQL(stmt *sqlite.Stmt, param string) {
	stmt.SetInt64(param, int64(e))
}

func EventFromSQL(stmt *sqlite.Stmt, param string) Event {
	n := stmt.GetInt64(param)
	if n > int64(^Event(0)) {
		panic(errors.New("Event cannot be greater than 8 bits"))
	}
	return Event(n)
}
