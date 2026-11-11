package main

import "testing"

// TestTheSelfTestDiscriminates runs the specimens and SN-03's controls as a
// unit test, so a rule that stops firing, or fires on a good form, fails
// `go test` as well as `make gate-controls`.
func TestTheSelfTestDiscriminates(t *testing.T) {
	if bad := runSelfTest(); bad != 0 {
		t.Fatal("the authoring self-test failed; its output above names each specimen")
	}
}
