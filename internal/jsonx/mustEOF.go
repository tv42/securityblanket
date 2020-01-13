package jsonx

import (
	"encoding/json"
	"fmt"
	"io"
)

// MustEOF returns an error if there is still (non-whitespace) content
// left.
func MustEOF(dec *json.Decoder) error {
	t, err := dec.Token()
	switch err {
	case io.EOF:
		// expected
		return nil
	case nil:
		return fmt.Errorf("invalid character after top-level value: %q", t)
	default:
		return err
	}
}
