package honeywell5800_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"eagain.net/go/securityblanket/internal/honeywell5800"
)

func TestChannelUnmarshalJSONLarge(t *testing.T) {
	var ch honeywell5800.Channel
	err := json.Unmarshal([]byte(fmt.Sprint(1<<4+1)), &ch)
	if !errors.Is(err, honeywell5800.ErrChannelTooLarge) {
		t.Errorf("wrong error: %v", err)
	}
}
