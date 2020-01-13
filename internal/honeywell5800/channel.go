package honeywell5800

import (
	"encoding/json"
	"errors"

	"crawshaw.io/sqlite"
)

var (
	ErrChannelTooLarge = errors.New("Channel cannot be larger than 4 bits")
)

type Channel uint8

// 4 bits on the air
const channelMax = 16

var _ json.Unmarshaler = (*Channel)(nil)

func (ch *Channel) UnmarshalJSON(data []byte) error {
	var n uint8
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	if n > channelMax {
		return ErrChannelTooLarge
	}
	*ch = Channel(n)
	return nil
}

func (ch Channel) ToSQL(stmt *sqlite.Stmt, param string) {
	stmt.SetInt64(param, int64(ch))
}
