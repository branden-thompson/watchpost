package tty

// setup_cast.go — the WATCHPOST RADIO - CORRESPONDENTS group (0.14.0 P4 Task
// 4.6; FR-4, MVS-D-25/27).
//
// A two-way radio: one voice for everything, or a cast. The cast is a MODE —
// switching back to Single Voice keeps the assignments unread rather than
// clearing them, so switching forward restores the whole cast (MVS-D-25).

import "github.com/branden-thompson/watchpost/platform/render"

// CastView is the cast as the TUI holds it: the mode and one name per role
// key. An absent or empty name means "inherit".
//
// Names, not pairs: this platform's half only. The app maps to and from the
// typed config, so the other operating system's half is never touched here
// (FR-8) — modes/tty does not know there is another one.
type CastView struct {
	Mode  string            // "" single voice | "cast"
	Names map[string]string // role key -> the voice assigned to it
}

// The cast modes, as the config spells them.
const (
	castModeSingle = ""
	castModeOn     = "cast"
)

// ToneState is the per-class mute, as the TUI holds it.
type ToneState struct {
	Mode  string   // "" all tones on | "mute"
	Muted []string // class keys
}

// voices is the host's voice list, or nothing when there is no hook.
func (d Dashboard) voices() []string {
	if d.cfg.Voices == nil {
		return nil
	}
	return d.cfg.Voices()
}

// voiceInstalled reports whether a voice is on this host. With no hook every
// voice reads as installed — a note nobody can act on is worse than none.
func (d Dashboard) voiceInstalled(name string) bool {
	return d.cfg.VoiceInstalled == nil || d.cfg.VoiceInstalled(name)
}

// castRowOrder is the five correspondent rows, in the order they are drawn.
func castRowOrder() []setupRowID {
	return []setupRowID{rowCastAlerts, rowCastWeather, rowCastMaritime, rowCastFire, rowCastSeismic}
}

// castLabel is what each row is called. The mock's two misspellings
// ("Maritme", "Hostpots") ship corrected.
func castLabel(id setupRowID) string {
	switch id {
	case rowCastAlerts:
		return "Alerts / Takeovers"
	case rowCastWeather:
		return "Location Report"
	case rowCastMaritime:
		return "Marine Report"
	case rowCastFire:
		return "Fire/Hotspots"
	case rowCastSeismic:
		return "Seismic Reports"
	}
	return ""
}

// roleOf is the cast role key a row assigns.
func roleOf(id setupRowID) string { return setupTable()[id].role }

// castLines draws the group: five rows, each a thing a listener hears and the
// voice that reads it.
//
// No mode radio and no enabling checkbox. Both made
// a listener combine two controls to know one answer — is this report going to
// be read by this voice? — where the row can simply say so. A row with no
// assignment of its own shows the voice it INHERITS, so every row names whoever
// will actually speak.
func (d Dashboard) castLines(o render.Opts) []string {
	chips := newArrowChips(o) // built once, shared by all five pickers
	lines := make([]string, 0, len(castRowOrder()))
	for _, id := range castRowOrder() {
		focused := d.setup.focus == id
		lines = append(lines, castRowIndent+setupMark(o, focused)+
			settingLabel(render.PadTo(castLabel(id), rowControlW), focused)+
			pickerCell(d.pickerName(id), chips, d.pickerFlashFor(id)))
	}
	return lines
}

// pickerCol is the column every picker starts in — wide enough for the longest
// label the group can draw ("Single Voice  (Default)" with its mark and radio),
// so nothing ever pushes a picker out of the column.
const (
	pickerCol  = 30
	castLabelW = 19 // the label column inside it, padded before tinting
)

// rowControlW is what a row spends before its ←→ control: the label column and
// the air after it, in ONE value.
//
// The tone toggles and the correspondent pickers read it both, so the two groups
// begin their controls in the same cell. They did not: the tone labels carried
// a cell more air, and with the balanced columns putting both groups in the same
// column that cell was visible as a step between them (HUM LEAD, UAT
// 2026-08-30). Two groups drawing the same shape must not each decide its
// geometry.
const rowControlW = castLabelW + 2

// pickerName is what a picker DISPLAYS: the voice that will actually speak.
//
// A row showing an empty box when the role inherits would be lying about what
// the listener hears — they would read "nothing" where a voice is going to
// read. So an unassigned row shows the name it INHERITS, which for every row
// in this group is the ROOT voice: the one that reads anything nobody has
// assigned, including the station's own lead and sign-off.
func (d Dashboard) pickerName(id setupRowID) string {
	if name := d.setup.cast.Names[roleOf(id)]; name != "" {
		return name
	}
	return d.rootDisplayName()
}

// rootDisplayName is the voice an unassigned row inherits.
//
// DISPLAY ONLY. When the config names no root the box shows the host's first
// voice so the row is not blank — but a save writes what the listener CHOSE,
// never the name the box happened to show them. Seeding the value would
// silently assign a voice nobody picked.
func (d Dashboard) rootDisplayName() string {
	if name := d.setup.cast.Names[roleRoot]; name != "" {
		return name
	}
	if v := d.voices(); len(v) > 0 {
		return v[0]
	}
	return "—"
}

// inheritEntry is the head of every picker's list: choosing it clears the
// role's own name so the role inherits again.
//
// It exists so ← is never a dead key. Without it a picker sitting on an
// inherited name has nothing to its left, and a listener pressing ← would
// conclude the control is broken.
const inheritEntry = "(inherit)"

// pickerList is the entries a picker cycles: the inherit entry first for any
// role that can inherit, then the host's voices.
func (d Dashboard) pickerList(setupRowID) []string {
	return append([]string{inheritEntry}, d.voices()...)
}

// cyclePicker moves a picker one entry left or right.
//
// It cycles BY THE ENTRY THE PICKER HOLDS, not by the name it displays: a row
// showing an inherited name sits on the inherit entry, so → assigns the first
// voice and ← wraps to the last. Cycling by the displayed name would jump to
// wherever the inherited voice happens to sit in the list, which is not where
// the listener is looking.
func (d Dashboard) cyclePicker(id setupRowID, forward bool) Dashboard {
	// THE RELAY ROW IS A PICKER WITHOUT A ROLE. Everything below cycles a voice
	// against the cast; this one cycles a duration against nothing, so it is
	// answered here rather than by giving the rotation a fake role to satisfy a
	// shape it does not have.
	if id == rowRelayLang {
		d.setup.relayLang = cycleRelayLang(d.setup.relayLang, forward)
		return d
	}
	if id == rowRelayDwell {
		d.setup.relayDwell = cycleRelayDwell(d.setup.relayDwell, forward)
		return d
	}
	list := d.pickerList(id)
	if len(list) == 0 {
		return d
	}
	role := roleOf(id)
	held := d.setup.cast.Names[role]
	if held == "" {
		held = inheritEntry
	}
	at := 0
	for i, e := range list {
		if e == held {
			at = i
			break
		}
	}
	step := 1
	if !forward {
		step = -1
	}
	next := list[((at+step)%len(list)+len(list))%len(list)]
	return d.assignRole(role, next)
}

// assignRole sets or clears a role's voice.
//
// Choosing a voice for a role TURNS THE CAST ON: under Single Voice the pairs
// are kept but unread (MVS-D-25), so a pick that left the mode alone would be
// silent — the listener would hear no change and reasonably conclude it broke.
func (d Dashboard) assignRole(role, entry string) Dashboard {
	names := make(map[string]string, len(d.setup.cast.Names)+1)
	for k, v := range d.setup.cast.Names {
		names[k] = v
	}
	if entry == inheritEntry || entry == "" {
		delete(names, role)
	} else {
		names[role] = entry
	}
	d.setup.cast.Names = names
	// The mode is DERIVED now that the window has no Single Voice / Cast
	// radio: any assignment means a cast is in force, and clearing the last
	// one means it is not. The key stays in the config — an older binary and
	// a hand-edited file both still read it — but a listener never sets it
	// directly, because they were being asked to operate a switch whose effect
	// the rows already showed.
	d.setup.cast.Mode = castModeSingle
	for _, r := range castRowOrder() {
		if names[roleOf(r)] != "" {
			d.setup.cast.Mode = castModeOn
			break
		}
	}
	return d.castTouched()
}

// castForSave is what a save writes: the mode, and only the names the rows
// actually showed as assigned. An override whose checkbox is unticked is
// CLEARED rather than carried, so the file matches the window.
func (d Dashboard) castForSave() CastView {
	out := CastView{Mode: d.setup.cast.Mode, Names: map[string]string{}}
	for role, name := range d.setup.cast.Names {
		if name != "" && name != inheritEntry {
			out.Names[role] = name
		}
	}
	return out
}

// castNote is the note under the focused row: why this row will not do what it
// looks like it will, or what it costs. Empty draws nothing.
//
// The DECK owns the install and failure words — one owner — so a failure's
// reason is never overwritten by a note assembled here.
func (d Dashboard) castNote(id setupRowID) string {
	if d.setup.note != "" && d.setup.noteRow == id {
		return d.setup.note // the deck's own words: progress, or why it failed
	}
	name := d.setup.cast.Names[roleOf(id)]
	switch {
	case len(d.voices()) == 0:
		return "No voices are available on this host."
	case name == "":
		return "Inherits " + d.pickerName(id) + "."
	case !d.voiceInstalled(name):
		// NO SIZE here. The note has 46 cells under a 50-cell row and the size
		// took it to 47, so it wrapped and left "(~63MB)" dangling on a line of
		// its own — and a longer voice name pushes it further (HUM LEAD, UAT
		// 2026-08-30). The size belongs where a listener is about to spend it:
		// the `p` offer below asks before downloading, and `report --verbose`
		// carries it too. This row's job is to say the voice is not here yet.
		return name + " not installed - downloads on save"
	case len(name) > pickerNameW:
		return name // truncated in the box; given in full here
	}
	return ""
}

// voiceDownloadSize is how big a voice is, in the one place the window says it:
// the `p` offer, which is where a listener consents to spending it. Named rather
// than written inline because it is a FACT about the model, not a phrase — the
// day the models change, this is the line to find.
const voiceDownloadSize = "~63MB"

// previewOffer is the ask-once flow (FR-4): on a voice that is not installed
// the FIRST p offers the download and the SECOND proceeds. A preview must not
// quietly pull a voice because somebody pressed a key to hear one.
func (d Dashboard) previewOffer(id setupRowID) (Dashboard, bool) {
	name := d.pickerName(id)
	if name == "" || name == "—" || d.cfg.PreviewVoice == nil {
		return d, false
	}
	installed := d.voiceInstalled(name)
	if installed || d.setup.offered == name {
		d.setup.offered = ""
		return d.settled(), true
	}
	d.setup.offered = name
	d.setup.note = name + " not installed - press p again to download (" + voiceDownloadSize + ")"
	d.setup.noteRow = id
	return d.settled(), false
}
