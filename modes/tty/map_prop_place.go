package tty

// map_prop_place.go — a reader's place in the Propagation mode's words
// (D-211, BUILD red team AX-5). A refresh, an hour step or an answer landing
// adds, drops and rewraps sentences above the scroll; a reader kept at the
// same line number would find another sentence there. The window remembers
// the sentence at its top, by its kind, and keeps it at the top wherever it
// moved, until the reader scrolls: a target that is being worked out again
// after a refresh is found again once it is answered.

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// wordsPlace is where a reader is in the words: the kinds of the sentence at
// the window's top, and how many of its rows are scrolled past.
type wordsPlace struct {
	keys   []string
	within int
}

// wordsShown reports whether the window shows the Propagation mode's words
// in place of the picture: the only time a place is kept.
func (d Dashboard) wordsShown() bool {
	return d.modal == modalMap && d.mapMode() == modePropagation && !d.mapPicture() && !d.mapPane.menuOn
}

// wordSentencesAll are the window's sentences in the Propagation mode, with
// their kinds: the mode's, then the place's heading and the layers' (D-210).
func (d Dashboard) wordSentencesAll() []wordSentence {
	all := d.describeLinesAll()
	loc := d.mapPlace()
	if loc == nil {
		return nil
	}
	out := d.propSentences(loc.Lat, loc.Lon)
	for i, l := range all[min(len(out), len(all)):] {
		out = append(out, said("after:"+strconv.Itoa(i), l)) // the heading, then each layer's sentence
	}
	return out
}

// sentenceStarts are the rows of the window's lines each sentence begins at,
// in order - a row that begins a sentence is the start of its text, however
// the window wrapped it - and -1 for one not found.
func (d Dashboard) sentenceStarts(lines []string, sentences []wordSentence) []int {
	starts := make([]int, len(sentences))
	from := 0
	for i, s := range sentences {
		starts[i] = -1
		for r := from; r < len(lines); r++ { // bounded by the window's lines (P10-02)
			if row := strings.TrimSpace(render.Plain(lines[r])); row != "" && strings.HasPrefix(s.text, row) {
				starts[i], from = r, r+1
				break
			}
		}
	}
	return starts
}

// sentenceRows is how many rows a sentence found at a start takes: to the
// next sentence found, or the end.
func sentenceRows(starts []int, i, lines int) int {
	for _, next := range starts[i+1:] {
		if next >= 0 {
			return next - starts[i]
		}
	}
	return lines - starts[i]
}

// readersScroll reports whether a message is the reader's own scroll.
func (d Dashboard) readersScroll(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || d.modal != modalMap {
		return false
	}
	act, bound := d.mapKeys.Lookup(key.String())
	return bound && (act == actMapScrollUp || act == actMapScrollDown)
}

// wordsPlaceNow is the reader's place before a message is handled: the one
// kept since the reader last scrolled - held while the words grow short
// enough to need no scroll, and found again when they grow back - else the
// sentence at the top now; false when none is kept: the words not shown, at
// their top, or the message the reader's own scroll.
func (d Dashboard) wordsPlaceNow(msg tea.Msg) (wordsPlace, bool) {
	if !d.wordsShown() || d.readersScroll(msg) {
		return wordsPlace{}, false
	}
	if kept := d.mapPane.wordsAt; len(kept.keys) > 0 {
		return kept, true
	}
	if d.modalScroll == 0 {
		return wordsPlace{}, false
	}
	return d.wordsTop()
}

// wordsTop is the sentence at the window's top and the rows of it scrolled
// past.
func (d Dashboard) wordsTop() (wordsPlace, bool) {
	sentences := d.wordSentencesAll()
	starts := d.sentenceStarts(d.modalLines(), sentences)
	top := -1
	for i, at := range starts {
		if at >= 0 && at <= d.modalScroll {
			top = i
		}
	}
	if top < 0 {
		return wordsPlace{}, false
	}
	return wordsPlace{keys: sentences[top].keys, within: d.modalScroll - starts[top]}, true
}

// followReader keeps the reader's place after a message (D-211): after
// their own scroll, the place they scrolled to; after anything else, the
// place they were at, put back at the top.
func (d Dashboard) followReader(msg tea.Msg, place wordsPlace, keep bool) Dashboard {
	switch {
	case !d.wordsShown():
		d.mapPane.wordsAt = wordsPlace{} // the words not shown: nothing to keep
	case d.readersScroll(msg) && d.modalScroll == 0:
		d.mapPane.wordsAt = wordsPlace{} // scrolled to the top: the reader is at the start
	case d.readersScroll(msg):
		top, ok := d.wordsTop()
		if !ok {
			top = wordsPlace{} // no sentence at the top to keep
		}
		d.mapPane.wordsAt = top
	case keep:
		if anchor := d.wordsAnchorNow(); anchor != d.mapPane.wordsAnchored {
			d = d.keepWordsPlace(place) // the words may have moved: find the place again
			d.mapPane.wordsAnchored = anchor
		}
		d.mapPane.wordsAt = place
	}
	return d
}

// wordsAnchor is what the words' layout is made from: the map's generation,
// which every answer, refresh and hour step raises, the window's size, and
// the minute, which the words' ages count in. While it stands the place kept
// is where it was, so a message that changes none - the visualiser's tick -
// lays out no words (D-211).
type wordsAnchor struct {
	gen    uint64
	w, h   int
	minute time.Time
}

// wordsAnchorNow is the words' anchor as the dashboard stands.
func (d Dashboard) wordsAnchorNow() wordsAnchor {
	return wordsAnchor{gen: d.mapPane.gen, w: d.width, h: d.height, minute: d.now().Truncate(time.Minute)}
}

// keepWordsPlace puts the sentence the reader was at back at the window's
// top: by its own kind first - a target by its first name - then by any
// kind it shared, else the scroll is left as it is.
func (d Dashboard) keepWordsPlace(p wordsPlace) Dashboard {
	if !d.wordsShown() || len(p.keys) == 0 {
		return d
	}
	sentences := d.wordSentencesAll()
	lines := d.modalLines()
	starts := d.sentenceStarts(lines, sentences)
	primary := p.keys[0]
	if len(p.keys) > 1 {
		primary = p.keys[1] // a target line's first name, before the kind every target line shares
	}
	for _, want := range append([]string{primary}, p.keys...) {
		for i, s := range sentences {
			if starts[i] < 0 || !hasKey(s.keys, want) {
				continue
			}
			rows := sentenceRows(starts, i, len(lines))
			d.modalScroll = min(starts[i]+min(p.within, max(rows-1, 0)), max(len(lines)-d.modalMax(), 0))
			return d
		}
	}
	return d
}

// hasKey reports whether a sentence is of a kind.
func hasKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
