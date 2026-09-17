// The oracle's stub: what bin/go, bin/sh, bin/gofmt and every `go build -o`
// output are symlinks to in a scratch tree. It records the invocation and
// answers with the painted status; the decisions are gateoracle's, unit-tested
// there. It is built by the oracle at test time — uninstrumented, so the race
// detector does not pay for every one of the thousands of times make execs it.
package main

import (
	"os"

	"github.com/branden-thompson/watchpost/tools/gateoracle"
)

func main() { os.Exit(gateoracle.StubMain(os.Args)) }
