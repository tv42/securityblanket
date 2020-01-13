package honeywell5800_test

import (
	"fmt"
	"testing"

	"eagain.net/go/securityblanket/internal/honeywell5800"
)

func TestEventString(t *testing.T) {
	run := func(e honeywell5800.Event, want string) {
		fn := func(t *testing.T) {
			g := e.String()
			if g != want {
				t.Errorf("wrong stringer: %q != %q", g, want)
			}
		}
		t.Run(fmt.Sprintf("%#02x", uint32(e)), fn)
	}
	run(0x02, "0x02")
	run(0xa0, "0xa0")
}

func TestEventFormat(t *testing.T) {
	run := func(e honeywell5800.Event, want string) {
		fn := func(t *testing.T) {
			g := fmt.Sprintf("%+v", e)
			if g != want {
				t.Errorf("wrong format: %q != %q", g, want)
			}
		}
		t.Run(fmt.Sprintf("%#02x", uint32(e)), fn)
	}
	run(0x02, "register")
	run(0xa0, "L1+L2")
	run(0x84, "L1+heartbeat")
	run(0x80, "L1")
	run(0x04, "heartbeat")
	run(0x8c, "L1+battery+heartbeat")
	// not seen with real hardware, just here to tease out coverage
	run(0x81, "L1+0x01")
}
