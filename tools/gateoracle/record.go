package gateoracle

import (
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// An invocation is `key#n`: the n-th time this run invoked the key. Keys are
// `go:test`, `go:mod:tidy`, `go:run:<package>`, `scripts/x.sh`, `built:<path>`,
// a tool's name, and `ctl:` in front of any of them run with `--self-test`.

func keyOf(inv string) string {
	if i := strings.LastIndex(inv, "#"); i >= 0 {
		return inv[:i]
	}
	return inv
}

func keysOf(invs []string) []string {
	var out []string
	for _, inv := range invs { // bounded by the reach (P10-02)
		if k := keyOf(inv); !contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

func countsOf(invs []string) map[string]int {
	out := map[string]int{}
	for _, inv := range invs { // bounded by the reach (P10-02)
		out[keyOf(inv)]++
	}
	return out
}

// encodeKey is a filename for a key; decodeKey is its inverse. `/` becomes
// `%2F` and `%` becomes `%25`, so no two keys share a file.
func encodeKey(k string) string { return url.PathEscape(k) }

func decodeKey(name string) string {
	k, err := url.PathUnescape(name)
	if err != nil {
		return name
	}
	return k
}

// recordInvocation appends `key#n` to the log under a file lock and returns it.
func recordInvocation(logPath, key string) (string, error) {
	lock, err := os.OpenFile(logPath+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return "", err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}
	n := 0
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the record (P10-02)
		if strings.HasPrefix(line, key+"#") {
			n++
		}
	}
	inv := key + "#" + strconv.Itoa(n+1)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(inv + "\n")
	return inv, err
}

// readLog is the invocations a run recorded, deduplicated, in order.
func readLog(logPath string) []string {
	raw, _ := os.ReadFile(logPath)
	var out []string
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the record (P10-02)
		if line != "" && !contains(out, line) {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

func contains(hay []string, want string) bool {
	for _, h := range hay { // bounded by the slice (P10-02)
		if h == want {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]int) []string {
	var out []string
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
