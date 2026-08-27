package ndbc

import (
	"crypto/sha256"
	"testing"
)

// Quality pass Q3 (CQ-12): GetText hands this parser the cache's own
// slice; it must never write into it.
func TestGetTextCallersMustNotMutate(t *testing.T) {
	raw := []byte("#YY  MM DD hh mm WDIR WSPD GST  WVHT   DPD   APD MWD   PRES  ATMP  WTMP  DEWP  VIS PTDY  TIDE\n" +
		"#yr  mo dy hr mn degT m/s  m/s     m   sec   sec degT   hPa  degC  degC  degC  nmi  hPa    ft\n" +
		"2026 08 24 01 00 270  5.0  6.0   1.2    12   8.0 260 1013.0  20.1  19.5  MM  MM  MM  MM\n")
	before := sha256.Sum256(raw)
	if _, err := ParseRealtime(raw); err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(raw) != before {
		t.Fatal("ParseRealtime wrote into the body it was handed (httpx.GetText contract)")
	}
}
