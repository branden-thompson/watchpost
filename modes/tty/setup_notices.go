package tty

// setup_notices.go — Settings' notice area (D-237): what a setting costs, how
// much the window's data holds, what an action did - at the bottom of every
// tab, above the controls, a row each in the credits' shape: its mark and
// label, " - ", and the words wrapped under their own column.

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
)

// noticeTone is what kind of thing a notice says, which its mark's colour
// tells.
type noticeTone int

const (
	noticeInfo noticeTone = iota // a fact: a size, a rate, a place
	noticeWarn                   // a cost to weigh before choosing
	noticeDone                   // an action's outcome
	noticeFail                   // an action that did not finish
)

// settingsNotice is one notice: the setting it is about, and its words.
type settingsNotice struct {
	label, text string
	tone        noticeTone
}

// noticeSlot is a place in a tab's notice area: the notice it shows now, if
// any, and the longest words it can hold, which the area keeps room for so
// nothing moves as notices come and go.
type noticeSlot struct {
	notice settingsNotice
	shown  bool
	room   string
}

// noticeIndent is the air before a notice's mark, the credits' (D-235).
const noticeIndent = creditIndent

// noticeRows is a notice in width cells: its mark and label in the tone's
// colour in a column of labelW, " - ", and its words wrapped under their own
// first line. Measured plain, so the colours never move the column.
func noticeRows(n settingsNotice, width, labelW int) []string {
	mark := "! " + n.label
	if code := n.tone.code(); code != "" {
		mark = render.Tint(mark, code)
	}
	lead := strings.Repeat(" ", noticeIndent) + mark + strings.Repeat(" ", max(0, labelW-render.Width(n.label))) + " - "
	leadW := noticeIndent + 2 + max(labelW, render.Width(n.label)) + 3
	words := render.WrapText(n.text, max(20, width-leadW))
	out := make([]string, 0, len(words))
	for i, w := range words { // the wrap's lines (P10-02)
		if i == 0 {
			out = append(out, lead+w)
			continue
		}
		out = append(out, strings.Repeat(" ", leadW)+w)
	}
	return out
}

// code is the tone's colour; a fact keeps the window's own.
func (t noticeTone) code() string {
	switch t {
	case noticeWarn:
		return render.Tok(render.AlertLabel)
	case noticeDone:
		return render.Tok(render.ProviderOK)
	case noticeFail:
		return render.Tok(render.AlertDanger)
	}
	return ""
}

// setupNotices is the open tab's notice area, a blank setting it off the
// groups, in the window's width w, and how many of its first rows are held
// room: as tall as the most its notices can say, whatever they say now, the
// notices at its foot over the controls - so neither the groups above nor the
// controls below move as notices come and go. A tab with nothing to say has
// no area.
func (d Dashboard) setupNotices(o render.Opts, w int) (area []string, held int) {
	slots := d.tabNoticeSlots()
	width := min(o.Width, w) - 5 // the borders and the panel's inset, as wrapModal's, and a cell of air at the right
	labelW := 0
	for _, s := range slots { // the tab's slots (P10-02)
		labelW = max(labelW, render.Width(s.notice.label))
	}
	var rows []string
	steady := 0
	for _, s := range slots { // the tab's slots (P10-02)
		steady += len(noticeRows(settingsNotice{label: s.notice.label, text: s.room}, width, labelW))
		if s.shown {
			rows = append(rows, noticeRows(s.notice, width, labelW)...)
		}
	}
	if steady == 0 && len(rows) == 0 {
		return nil, 0
	}
	held = max(0, steady-len(rows))
	area = make([]string, held+1, held+1+len(rows)) // the held room, then the blank over the notices
	if len(rows) == 0 {
		held++ // nothing to set off: the blank is held room too
	}
	return append(area, rows...), held
}

// tabNoticeSlots are the open tab's places for notices.
func (d Dashboard) tabNoticeSlots() []noticeSlot {
	var slots []noticeSlot
	switch d.setupTab() {
	case tabMaps:
		slots = d.mapNoticeSlots()
	case tabData:
		slots = []noticeSlot{d.historyNoticeSlot()}
	case tabRadio:
		slots = []noticeSlot{d.castNoticeSlot()}
	}
	kept := slots[:0]
	for _, s := range slots { // the tab's slots (P10-02)
		if s.room != "" {
			kept = append(kept, s)
		}
	}
	return kept
}

// slotOf is a slot showing a notice when shown and it has words, kept wide
// enough for the widest of the words it can hold.
func slotOf(n settingsNotice, shown bool, could ...string) noticeSlot {
	s := noticeSlot{notice: n, shown: shown && n.text != ""}
	for _, c := range append(could, n.text) { // the slot's possible words (P10-02)
		if render.Width(c) > render.Width(s.room) {
			s.room = c
		}
	}
	return s
}

// mapNoticeSlots are the Maps tab's: what Open-Meteo bills, the UV cities'
// asks while the row is chosen, the layers' cost, and how long the map keeps
// its data - or what clearing it did.
func (d Dashboard) mapNoticeSlots() []noticeSlot {
	head, detail := costWarningParts(d.mapCost)
	worstHead, worstDetail := costWarningParts(worstMapCost)
	return []noticeSlot{
		slotOf(settingsNotice{"Temperature", meteredNote, noticeWarn}, !d.mapTempNDFD),
		slotOf(settingsNotice{"Rain Day 4+", rainFullNote, noticeWarn}, d.mapRainFull),
		slotOf(settingsNotice{"UV cities", uvCitiesNote(d.mapUVCities), noticeInfo}, d.setup.focus == rowMapUVCities, uvCitiesNote(uvCityChoices[len(uvCityChoices)-1])),
		slotOf(settingsNotice{"Map layers", strings.TrimSpace(head + " " + detail), noticeWarn}, true, worstHead+" "+worstDetail),
		d.outcomeSlot("Map data", d.cfg.MapRetention, rowMapClear, mapClearedNote(MapCleared{Files: 99999, Zones: 99999})),
	}
}

// worstMapCost is the largest estimate the layers' notice is kept room for.
var worstMapCost = MapCost{Bytes: 999_900_000, Requests: 99_999}

// historyNoticeSlot is the Data tab's: how much the history holds and where,
// or what clearing it did.
func (d Dashboard) historyNoticeSlot() noticeSlot {
	usage := ""
	if d.cfg.HistoryUsage != nil {
		usage = d.cfg.HistoryUsage()
	}
	return d.outcomeSlot("History", usage, rowHistoryClear, historyClearedNote)
}

// outcomeSlot is a row's standing fact, or the outcome of its action while
// that is the latest thing the window has to say about it.
func (d Dashboard) outcomeSlot(label, fact string, row setupRowID, outcome string) noticeSlot {
	n := settingsNotice{label: label, text: fact}
	if d.setup.note != "" && d.setup.noteRow == row {
		n.text, n.tone = d.setup.note, d.setup.noteTone
	}
	return slotOf(n, true, fact, outcome)
}

// castNoticeSlot is the Radio tab's: the focused voice row's note - why it
// will not do what it looks like it will, or what it costs.
func (d Dashboard) castNoticeSlot() noticeSlot {
	focus := d.setup.focus
	n := settingsNotice{label: "Voice"}
	if setupTable()[focus].group == groupCast {
		n.text = d.castNote(focus)
		if name := d.setup.cast.Names[roleOf(focus)]; name != "" && !d.voiceInstalled(name) {
			n.tone = noticeWarn
		}
	}
	longest := strings.Repeat("W", pickerNameW) + " not installed - press p again to download (" + voiceDownloadSize + ")"
	return slotOf(n, true, longest)
}

// mapClearedNote says what "Clear map data" removed, under the Map data
// label.
func mapClearedNote(r MapCleared) string {
	return "Cleared: " + strconv.Itoa(r.Files) + " tile files and " + strconv.Itoa(r.Zones) + " zone outlines removed."
}
