package geodata

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// IS-6 (red-team-discover #38, due at the milestone that embeds datasets):
// the embedded payloads are integrity-pinned. Regenerating the data is an
// INTENTIONAL act: re-run the trim pipeline (tools/geotrim/refresh.sh),
// update these hashes in the same commit, and say so in the message.
const (
	citiesSHA256 = "db2722a571872504bdb10554dc10bf342f233c6260405b54a96ed8a6a72a71a5"
	zipsSHA256   = "ca5ea42c7419662954e97a8a5ec21c62ad9b710d9cef3a0de860f47634650921"
)

func TestEmbeddedDataIntegrity(t *testing.T) {
	for name, tc := range map[string]struct {
		data []byte
		want string
	}{
		"cities": {citiesGz, citiesSHA256},
		"zips":   {zipsGz, zipsSHA256},
	} {
		sum := sha256.Sum256(tc.data)
		if got := hex.EncodeToString(sum[:]); got != tc.want {
			t.Errorf("%s payload hash %s != pinned %s — if the regeneration was intentional, update the pin in the same commit", name, got, tc.want)
		}
	}
}
