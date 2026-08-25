package synth

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// UGC filtering (UAT 81): an office's product carries every zone it
// serves, "$$"-separated, each block headed by a UGC line such as
// "CAZ552-251015-" or "CAZ043>048-055-251015-". The broadcast reads the
// blocks covering the location's forecast zone or county — never a
// neighbour's — plus the product's preamble (title, office, issuance).

var ugcLine = regexp.MustCompile(`^([A-Z]{2}[CZ]\d{3}[-\d>A-Z]*)-\d{6}-$`)

// FilterUGC keeps the preamble and the blocks whose UGC set contains zone
// or county. A product with no UGC lines is returned unchanged.
func FilterUGC(text, zone, county string) string {
	blocks := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "$$")
	var out []string
	sawUGC := false
	for i, block := range blocks {
		pre, codes, body, ok := splitUGC(block)
		if !ok {
			if i == 0 {
				out = append(out, block) // preamble-only first block
			}
			continue
		}
		sawUGC = true
		if i == 0 && strings.TrimSpace(pre) != "" {
			out = append(out, pre)
		}
		if codes[zone] || codes[county] {
			out = append(out, body)
		}
	}
	if !sawUGC {
		return text
	}
	return strings.Join(out, "\n\n")
}

// splitUGC finds the block's UGC line; returns the text before it, the
// expanded code set, and the text from the UGC line on.
func splitUGC(block string) (pre string, codes map[string]bool, body string, ok bool) {
	lines := strings.Split(block, "\n")
	for i, l := range lines {
		l = strings.TrimSpace(l)
		m := ugcLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		return strings.Join(lines[:i], "\n"), expandUGC(m[1]), strings.Join(lines[i:], "\n"), true
	}
	return "", nil, block, false
}

// expandUGC turns "CAZ043>048-055-CAC073" into the set of codes it names.
func expandUGC(spec string) map[string]bool {
	codes := map[string]bool{}
	prefix := ""
	for _, part := range strings.Split(spec, "-") {
		if part == "" {
			continue
		}
		if len(part) >= 4 && part[2] == 'C' || len(part) >= 4 && part[2] == 'Z' {
			prefix = part[:3]
			part = part[3:]
		}
		if prefix == "" {
			continue
		}
		lo, hi := part, part
		if i := strings.Index(part, ">"); i >= 0 {
			lo, hi = part[:i], part[i+1:]
		}
		a, err1 := strconv.Atoi(lo)
		b, err2 := strconv.Atoi(hi)
		if err1 != nil || err2 != nil || b < a || b-a > 999 {
			continue
		}
		for n := a; n <= b; n++ {
			codes[prefix+padZone(n)] = true
		}
	}
	return codes
}

func padZone(n int) string { return fmt.Sprintf("%03d", n) }
