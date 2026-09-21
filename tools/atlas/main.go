// Command atlas builds the architecture atlas from the tracked documents.
//
// **The documents are the source of truth and the page is generated from
// them.** It reads every fenced mermaid block under 06_docs/ and docs/, takes
// the heading above each as its title and the paragraph between as its lead,
// and groups them by the feature they belong to.
//
// Two things worth knowing before editing a diagram and wondering why the page
// did not change: only fenced mermaid blocks are read, so **a change written in
// the prose beside a diagram never reaches the atlas**; and a feature not named
// in the reading order below is still built, after the named ones, so nothing
// is silently dropped.
//
// Run from the repository root: go run ./tools/atlas
package main

import (
	_ "embed"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed template.html
var page string

// order is what a reader should meet first, not alphabetical.
var order = []struct{ dir, title string }{
	{"watchpost-cli", "The shape of the program"},
	{"map-ready-geometry", "Map-ready geometry (0.17.0)"},
	{"severe-alerts-modals", "Severe weather"},
	{"seismic-data", "Earthquakes"},
	{"global-ticker", "The national feed"},
	{"multi-voice-support", "The Director and the voices"},
	{"0.16.0-broadcaster-ui", "The Broadcaster console"},
	{"0.15.0-pre-broadcaster-ui-improvements", "Before the console"},
	{"watchpost-performance-quality-pass", "The quality pass"},
}

var (
	block   = regexp.MustCompile("(?s)```mermaid\n(.*?)```")
	heading = regexp.MustCompile(`(?m)^#+ (.+)$`)
	bold    = regexp.MustCompile(`\*\*(.+?)\*\*`)
	ital    = regexp.MustCompile(`\*(.+?)\*`)
	mono    = regexp.MustCompile("`([^`]+)`")
	blanks  = regexp.MustCompile(`\n+`)
)

type diagram struct{ feature, file, title, lead, src string }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "atlas:", err)
		os.Exit(1)
	}
}

func run() error {
	var found []diagram
	for _, root := range []string{"06_docs", "docs"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			found = append(found, read(path, string(raw))...)
			return nil
		})
		if err != nil {
			return err
		}
	}
	if len(found) == 0 {
		return fmt.Errorf("no mermaid blocks found")
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].file < found[j].file })

	seen := map[string]bool{}
	var areas []struct{ dir, title string }
	for _, o := range order {
		areas = append(areas, o)
		seen[o.dir] = true
	}
	var rest []string
	for _, d := range found {
		if !seen[d.feature] {
			seen[d.feature] = true
			rest = append(rest, d.feature)
		}
	}
	sort.Strings(rest)
	for _, r := range rest {
		areas = append(areas, struct{ dir, title string }{r, strings.ReplaceAll(r, "-", " ")})
	}

	var nav, body strings.Builder
	for _, a := range areas {
		var mine []diagram
		for _, d := range found {
			if d.feature == a.dir {
				mine = append(mine, d)
			}
		}
		if len(mine) == 0 {
			continue
		}
		fmt.Fprintf(&nav, `<li><a href="#%s">%s<span class="n">%d</span></a></li>`, a.dir, html.EscapeString(a.title), len(mine))
		fmt.Fprintf(&body, `<section class="level" id="%s"><header class="lvl"><h2>%s</h2></header>`, a.dir, html.EscapeString(a.title))
		for i, d := range mine {
			lead := ""
			if d.lead != "" {
				lead = `<p class="lead">` + d.lead + `</p>`
			}
			fmt.Fprintf(&body, `<article class="dia" id="%s-%d"><div class="meta"><h3>%s</h3>`+
				`<p class="file"><code>%s</code></p>%s</div>`+
				`<div class="canvas"><pre class="mermaid">%s</pre></div></article>`,
				a.dir, i, html.EscapeString(d.title), html.EscapeString(d.file), lead, html.EscapeString(d.src))
		}
		body.WriteString("</section>")
	}

	out := strings.NewReplacer(
		"<!--NAV-->", nav.String(),
		"<!--BODY-->", body.String(),
		"<!--COUNT-->", fmt.Sprint(len(found)),
	).Replace(page)
	dest := filepath.Join("06_docs", "architecture-atlas.html")
	if err := os.WriteFile(dest, []byte(out), 0o644); err != nil {
		return err
	}
	fmt.Printf("%d diagrams -> %s\n", len(found), dest)
	return nil
}

// read pulls every diagram out of one document.
func read(path, text string) []diagram {
	var out []diagram
	for _, m := range block.FindAllStringSubmatchIndex(text, -1) {
		heads := heading.FindAllStringSubmatchIndex(text[:m[0]], -1)
		title, lead := filepath.Base(path), ""
		if len(heads) > 0 {
			last := heads[len(heads)-1]
			title = strings.TrimSpace(text[last[2]:last[3]])
			lead = strings.TrimSpace(text[last[1]:m[0]])
			lead = blanks.ReplaceAllString(lead, " ")
			if strings.HasPrefix(lead, "|") || strings.HasPrefix(lead, "```") || len(lead) > 420 {
				lead = ""
			}
		}
		out = append(out, diagram{
			feature: featureOf(path),
			file:    filepath.ToSlash(path),
			title:   title,
			lead:    inline(lead),
			src:     text[m[2]:m[3]],
		})
	}
	return out
}

// featureOf is the area a document belongs to.
func featureOf(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "02_features" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "docs"
}

// inline turns the little markdown a lead may carry into html.
func inline(s string) string {
	if s == "" {
		return ""
	}
	s = html.EscapeString(s)
	s = bold.ReplaceAllString(s, "<strong>$1</strong>")
	s = ital.ReplaceAllString(s, "<em>$1</em>")
	return mono.ReplaceAllString(s, "<code>$1</code>")
}
