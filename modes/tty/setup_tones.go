package tty

// setup_tones.go — the ALERTS - TONE group (0.14.0 P4 Task 4.5; FR-11,
// MVS-D-26/28).
//
// Every alert class always sounds its own ratified preset; the only choice here
// is whether it sounds at all. So this is a MUTE, not a tone chooser — the
// wording, the marks and the rules all say so.

import "github.com/branden-thompson/watchpost/platform/render"

// ToneClass is one mutable class, as the app supplies it: the config key and
// the label the mock draws. modes/tty may not import the registry, so the list
// arrives as data and a parity test in app pins it.
type ToneClass struct {
	Key   string
	Label string
}

// classRowOrder is the six class rows in the mock's reading order.
func classRowOrder() []setupRowID {
	return []setupRowID{rowClassDisaster, rowClassWarning, rowClassWatch, rowClassAdvisory, rowClassStatement, rowClassStorm}
}

// ONE CLASS PER LINE, at every width.
//
// The classes were laid two abreast, and then two abreast with each sub-column
// sized to its own labels. Both are gone. Six rows of one thing each is a list,
// and a list is what this is: the eye runs down a single column of state words
// and reads the answer, where two columns made it scan across a gap and back.
//
// So there is no breakpoint here to keep in step with the window's, no
// second-column geometry, and no arrangement for the scroll to disagree with —
// a class's line IS its position in the order.

// toneLabelW is the class column: the widest label the app supplied, plus air.
// Measured rather than fixed, because the labels come from the registry and a
// constant would silently truncate or over-pad the next one added.
func toneLabelW(order []setupRowID, labels map[string]string) int {
	// Never narrower than a correspondent row's, so the two groups' controls
	// begin in the same cell — and still measured, so a label longer than that
	// widens every row together rather than pushing one row's control out of
	// line with the rest.
	w := rowControlW
	for _, id := range order {
		w = max(w, render.Width(labels[classKeyOf(id)])+toneLabelAir)
	}
	return w
}

// toneIndent is the group's left inset (the focus mark's cell included) and
// toneLabelAir the air between a label and its toggle.
const (
	toneIndent   = "  "
	toneLabelAir = 2
)

// toneLineOf is a class row's offset within the group's block.
func toneLineOf(order []setupRowID, id setupRowID) int {
	for i, r := range order {
		if r == id {
			return i
		}
	}
	return 0
}

// toneLines draws the group: every class on its own line, showing whether it
// will sound.
func (d Dashboard) toneLines(o render.Opts) []string {
	order, labels, chips := classRowOrder(), d.toneClassLabels(), newArrowChips(o)
	labelW := toneLabelW(order, labels)
	lines := make([]string, 0, len(order))
	for _, id := range order {
		// PAD FIRST, then tint: padding a tinted string makes PadTo measure
		// display width across the escapes, which is far dearer than padding
		// the plain text and colouring the result.
		focused := d.setup.focus == id
		lines = append(lines, toneIndent+setupMark(o, focused)+
			settingLabel(render.PadTo(labels[classKeyOf(id)], labelW), focused)+
			toggleCell(d.toneStateWord(id), chips, d.pickerFlashFor(id)))
	}
	return lines
}

// toneStateWord is what the row says: MUTED when this class will sound nothing,
// Enabled when it will.
//
// It is read through the SAME rule the ear follows (cast.Muted's shape, mirrored
// here because modes/tty may not import the registry), so the screen cannot say
// one thing while the broadcast does another — which is the whole reason the
// state is shown as a word rather than inferred from a box and a mode.
func (d Dashboard) toneStateWord(id setupRowID) string {
	if d.setup.classMuted(classKeyOf(id)) {
		return "MUTED"
	}
	return "Enabled"
}

// classMuted mirrors cast.Muted: mute mode AND (the set names the class, or the
// set is empty — "Mute with nothing ticked" means every class).
func (st setupState) classMuted(key string) bool {
	if st.toneMode != toneModeMute {
		return false
	}
	if len(st.toneMuted) == 0 {
		return true
	}
	return st.toneMuted[key]
}

// classKeyOf is the tone-class key a row mutes.
func classKeyOf(id setupRowID) string { return setupTable()[id].class }

// toneClassLabels maps each class key to its label, from the list the app
// supplied. A class the app did not name falls back to its key rather than
// drawing an empty cell — a row that draws nothing is worse than an ugly one.
func (d Dashboard) toneClassLabels() map[string]string {
	out := make(map[string]string, len(d.cfg.ToneClasses))
	for _, c := range d.cfg.ToneClasses {
		out[c.Key] = c.Label
	}
	for _, id := range classRowOrder() {
		if k := classKeyOf(id); out[k] == "" {
			out[k] = k
		}
	}
	return out
}

// toggleClass flips one class between Enabled and MUTED.
//
// It MATERIALISES the implicit set first, and that is the whole subtlety. Under
// mute mode an EMPTY set means every class (MVS-D-26) — so with the set empty
// and the listener enabling "Warnings", simply removing the key would leave the
// set empty and every class still muted, including the one they just enabled.
// The five that stay muted have to be written down before the sixth can leave.
//
// And when the last muted class is enabled the MODE goes back to "all tones
// on", so the rows cannot end up saying MUTED because of an empty set nobody
// asked for.
func (d Dashboard) toggleClass(id setupRowID) Dashboard {
	key := classKeyOf(id)
	if key == "" {
		return d
	}
	next := map[string]bool{}
	if d.setup.toneMode == toneModeMute {
		if len(d.setup.toneMuted) == 0 {
			for _, k := range classRowOrder() { // the implicit "all", written down
				next[classKeyOf(k)] = true
			}
		} else {
			for k, v := range d.setup.toneMuted {
				next[k] = v
			}
		}
	}
	if next[key] {
		delete(next, key)
	} else {
		next[key] = true
	}
	d.setup.toneMuted = next
	d.setup.toneMode = toneModeMute
	if len(next) == 0 {
		d.setup.toneMode = toneModeOn
	}
	return d.castTouched()
}

// The tone modes, as the config spells them (platform/config's closed set).
const (
	toneModeOn   = ""
	toneModeMute = "mute"
)

// mutedClassKeys is the ticked set, in the registry's order so a save and a
// reload produce the same file.
func (st setupState) mutedClassKeys() []string {
	var out []string
	for _, id := range classRowOrder() {
		if k := classKeyOf(id); st.toneMuted[k] {
			out = append(out, k)
		}
	}
	return out
}
