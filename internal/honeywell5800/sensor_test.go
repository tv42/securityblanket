package honeywell5800_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"eagain.net/go/securityblanket/internal/honeywell5800"
)

func TestSensorUnmarshalJSONLarge(t *testing.T) {
	var id honeywell5800.Sensor
	err := json.Unmarshal([]byte(fmt.Sprint(1<<20+1)), &id)
	if !errors.Is(err, honeywell5800.ErrSensorIDTooLarge) {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSensorString(t *testing.T) {
	id := honeywell5800.Sensor(643345)
	if g, e := id.String(), `A064-3345`; g != e {
		t.Errorf("wrong stringer: %q != %q", g, e)
	}
}
