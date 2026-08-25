package coops

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, raw []byte, out any) error {
	t.Helper()
	return json.Unmarshal(raw, out)
}
