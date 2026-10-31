// Command finaltag refuses a final release built on a paired library's
// pre-release: a tag of the form vX.Y.Z, with no suffix, while go.mod
// requires go-ionomaps or go-tuiMaps at a pre-release version (vX.Y.Z-rc.N),
// or replaces either with anything at all, fails, naming each (REVIEW red
// team RH-9, R2H-3; D-3: watchpost ships on the paired libraries' final
// tags). go.mod is read by the go tool's own parser (`go mod edit -json`),
// so its layout - a replace, CRLF line endings, a trailing space - cannot
// hide a requirement. Other modules' versions are their own projects'. A
// pre-release tag is not held to it.
//
//	go run ./tools/finaltag TAG [GO_MOD]
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// finalTag is a release tag with no pre-release suffix.
var finalTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// preRelease is a semantic version with a suffix: a release candidate, or a
// pseudo-version.
var preRelease = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-`)

var errUsage = errors.New("usage: finaltag TAG [GO_MOD]")

// module is a go.mod requirement or one side of a replace, as `go mod edit
// -json` writes it.
type module struct {
	Path, Version string
}

// goMod is the part of `go mod edit -json` this reads.
type goMod struct {
	Require []module
	Replace []struct{ Old, New module }
}

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
	raw, err := exec.Command("go", "mod", "edit", "-json", path).Output()
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "finaltag: go mod edit -json", path+":", err)
		return 2
	}
	var mod goMod
	if err := json.Unmarshal(raw, &mod); err != nil {
		_, _ = fmt.Fprintln(errOut, "finaltag: go mod edit -json wrote no module:", err)
		return 2
	}
	held := heldBack(args[0], mod)
	if len(held) > 0 {
		_, _ = fmt.Fprintf(errOut, "finaltag: %s is a final tag, and go.mod holds the paired libraries back: %s - tag the libraries' final versions and pin them first, with no replace (D-3)\n", args[0], strings.Join(held, ", "))
		return 1
	}
	_, _ = fmt.Fprintf(out, "finaltag: %s may be released on what go.mod requires\n", args[0])
	return 0
}

// paired are the libraries D-3 pairs with watchpost.
var paired = []string{"github.com/branden-thompson/go-ionomaps", "github.com/branden-thompson/go-tuimaps"}

// heldBack are the paired libraries' pre-releases, and their replacements, a
// final tag may not ship on; none for a pre-release tag.
func heldBack(tag string, mod goMod) []string {
	if !finalTag.MatchString(tag) {
		return nil
	}
	var out []string
	for _, r := range mod.Require {
		if slices.Contains(paired, r.Path) && preRelease.MatchString(r.Version) {
			out = append(out, r.Path+" "+r.Version)
		}
	}
	for _, r := range mod.Replace {
		if slices.Contains(paired, r.Old.Path) {
			out = append(out, r.Old.Path+" replaced by "+strings.TrimSpace(r.New.Path+" "+r.New.Version))
		}
	}
	return out
}
