// Command finaltag refuses a final release built on a paired library's
// pre-release: a tag of the form vX.Y.Z, with no suffix, while go.mod
// requires go-ionomaps or go-tuiMaps at a pre-release version (vX.Y.Z-rc.N)
// fails, naming each (REVIEW red team RH-9; D-3: watchpost ships on the
// paired libraries' final tags). Other modules' versions are their own
// projects'. A pre-release tag is not held to it.
//
//	go run ./tools/finaltag TAG [GO_MOD]
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
)

// finalTag is a release tag with no pre-release suffix.
var finalTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// preRelease is a required module at a pre-release version: its path, then
// a semantic version with a suffix.
var preRelease = regexp.MustCompile(`(?m)^\s*(?:require\s+)?(\S+)\s+(v[0-9]+\.[0-9]+\.[0-9]+-[0-9A-Za-z.-]+)(?:\s*//.*)?$`)

var errUsage = errors.New("usage: finaltag TAG [GO_MOD]")

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		_, _ = fmt.Fprintln(errOut, errUsage)
		return 2
	}
	path := "go.mod"
	if len(args) == 2 {
		path = args[1]
	}
	mod, err := os.ReadFile(path)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "finaltag:", err)
		return 2
	}
	held := heldBack(args[0], string(mod))
	if len(held) > 0 {
		_, _ = fmt.Fprintf(errOut, "finaltag: %s is a final tag, and go.mod requires pre-release modules: %s - tag the libraries' final versions and pin them first (D-3)\n", args[0], strings.Join(held, ", "))
		return 1
	}
	_, _ = fmt.Fprintf(out, "finaltag: %s may be released on what go.mod requires\n", args[0])
	return 0
}

// paired are the libraries D-3 pairs with watchpost.
var paired = []string{"github.com/branden-thompson/go-ionomaps", "github.com/branden-thompson/go-tuimaps"}

// heldBack are the paired libraries' pre-releases a final tag may not ship
// on; none for a pre-release tag. A pseudo-version is a pre-release too.
func heldBack(tag, mod string) []string {
	if !finalTag.MatchString(tag) {
		return nil
	}
	var out []string
	for _, m := range preRelease.FindAllStringSubmatch(mod, -1) {
		if slices.Contains(paired, m[1]) {
			out = append(out, m[1]+" "+m[2])
		}
	}
	return out
}
