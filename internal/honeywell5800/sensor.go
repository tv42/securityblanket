package honeywell5800

import (
	"encoding/json"
	"errors"
	"fmt"

	"crawshaw.io/sqlite"
)

var (
	ErrSensorIDTooLarge = errors.New("sensor ID cannot be larger than 20 bits")
	ErrSensorIDTooSmall = errors.New("sensor ID cannot be 0 or negative")
)

type Sensor uint32

// uses only 20 bits as per
// https://github.com/merbanan/rtl_433/blob/master/src/devices/honeywell.c
const sensorMax = 1 << 20

var _ json.Unmarshaler = (*Sensor)(nil)

func (s *Sensor) UnmarshalJSON(data []byte) error {
	var n uint32
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	if n > sensorMax {
		return ErrSensorIDTooLarge
	}
	if n <= 0 {
		return ErrSensorIDTooSmall
	}
	*s = Sensor(n)
	return nil
}

var _ fmt.Stringer = Sensor(0)

// String returns the sensor ID formatted like A000-0000, as found on
// stickers on the hardware.
func (s Sensor) String() string {
	str := fmt.Sprintf("A%07d", s)
	return str[:len(str)-4] + "-" + str[len(str)-4:]
}

func (s Sensor) ToSQL(stmt *sqlite.Stmt, param string) {
	stmt.SetInt64(param, int64(s))
}

func SensorFromSQL(stmt *sqlite.Stmt, param string) Sensor {
	col := stmt.ColumnIndex(param)
	if col < 0 {
		panic(fmt.Errorf("no such column in sql row: %q", param))
	}
	n := stmt.ColumnInt64(col)
	if n > sensorMax {
		panic(ErrSensorIDTooLarge)
	}
	if n <= 0 {
		panic(ErrSensorIDTooSmall)
	}
	return Sensor(n)
}
