package firms

import (
	"crypto/sha256"
	"testing"
)

// Quality pass Q3 (CQ-12): GetText hands this parser the cache's own
// slice; it must never write into it.
func TestGetTextCallersMustNotMutate(t *testing.T) {
	raw := []byte(csvBody)
	before := sha256.Sum256(raw)
	if _, err := ParseCSV(raw); err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(raw) != before {
		t.Fatal("ParseCSV wrote into the body it was handed (httpx.GetText contract)")
	}
}
