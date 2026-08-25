package components

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// CreateMessageBlock renders a colored marker followed by body text wrapped to
// the terminal width, with continuation lines indented under the first body
// character (hanging indent = DisplayWidth(marker) + 1). The canonical STUDS
// pattern for `!` warning, `?` question, and `i` informational message blocks —
// replaces hardcoded line breaks that ignored the actual terminal width.
//
// GO-STUDS single-function API pattern: width auto-detects via the canonical
// rendering.GetTerminalSize source; pass an explicit width as the optional
// final argument.
//
//	CreateMessageBlock("!", "93", "long warning text …")        // auto width
//	CreateMessageBlock("i", design.GetColor("color.cyan.bright"),
//	    "informational text …", 80)                             // explicit width
//
// Wrapping is ANSI-aware and runewidth-correct (rendering.WrapTextANSI): escape
// sequences in the body are excluded from width math, and wide glyphs count
// their true terminal cells. The marker color is classified basic-vs-256 via
// rendering.SGR; an empty markerColor renders the marker plain.
//
// The block always terminates with "\n\n" — the seam separator that keeps a
// following prompt line from colliding with the last wrapped line.
func CreateMessageBlock(marker, markerColor, body string, width ...int) string {
	w := 0
	if len(width) > 0 {
		w = width[0]
	} else {
		w, _ = rendering.GetTerminalSize()
	}

	indentWidth := rendering.DisplayWidth(marker) + 1
	indent := strings.Repeat(" ", indentWidth)

	// Body wraps within the space right of the hanging indent. A degenerate
	// wrap width (<= 0) falls through to WrapTextANSI's graceful-overflow
	// contract: the body renders unwrapped on one line.
	lines := rendering.WrapTextANSI(body, w-indentWidth)

	renderedMarker := marker
	if markerColor != "" {
		renderedMarker = rendering.WrapSGR(marker, markerColor)
	}

	var out strings.Builder
	for i, line := range lines {
		if i == 0 {
			out.WriteString(renderedMarker)
			out.WriteString(" ")
		} else {
			out.WriteString(indent)
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	out.WriteString("\n")
	return out.String()
}
