package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestEveryPickerNamesWhatItsArrowsChange is 0.19.5 Q9.3 (JM-8): a picker row
// that is not a voice names what ←→ change on it. Without its words the chip
// row falls back to a voice's, "←→ Voice  p Preview", on a row that has
// neither; the walkthrough's proof found that nothing else would say so.
func TestEveryPickerNamesWhatItsArrowsChange(t *testing.T) {
	for id, r := range setupTable() {
		if !r.picker || r.group == groupCast {
			continue
		}
		if _, ok := pickerChipWords[setupRowID(id)]; !ok {
			t.Errorf("picker row %d (group %d) has no words in pickerChipWords: its chips would offer a voice", id, r.group)
		}
	}
}

// TestTheRelayRowsOfferNoVoice holds what the operator sees on the relay
// replay's two rows: what their arrows change, and no preview key.
func TestTheRelayRowsOfferNoVoice(t *testing.T) {
	d := dash(t).(Dashboard)
	for _, c := range []struct {
		row  setupRowID
		word string
	}{{rowRelayDwell, "Rotation"}, {rowRelayLang, "Language"}} {
		d.setup.focus = c.row
		chips := strings.Join(d.setupChips(d.opts()), " ")
		if !strings.Contains(chips, c.word) || strings.Contains(chips, "Preview") || strings.Contains(chips, "Voice") {
			t.Errorf("row %d's chips read %q; want ←→ %s and no voice or preview", c.row, chips, c.word)
		}
	}
}

// TestAnArrowOnASavedSettingMarksItForSaving is 0.19.5 Q9.3 (JM-8): a row of
// the groups Settings saves on close, through uiForSave, marks the window for
// saving when its arrows move it. A change that returned without uiTouched
// would draw and then be lost when the window closes, and the walkthrough's
// proof found nothing else that said so. The tone, cast, relay and history
// groups save through their own hooks and are not this rule's.
func TestAnArrowOnASavedSettingMarksItForSaving(t *testing.T) {
	d, err := NewDashboard(Config{Voices: func() []string { return []string{"Samantha", "Daniel", "Karen"} }})
	if err != nil {
		t.Fatal(err)
	}
	d.width, d.height = 160, 60
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	saved := map[setupGroupID]bool{groupUI: true, groupMap: true, groupLayers: true, groupMapLayers: true}
	checked := 0
	for _, row := range setupTable() {
		if !saved[row.group] || row.kind != rowToggle && row.kind != rowPicker {
			continue
		}
		checked++
		for _, k := range []tea.KeyPressMsg{{Code: tea.KeyLeft}, {Code: tea.KeyRight}} {
			m, _ := d.openSetupAt(row.id).Update(k)
			if !m.(Dashboard).setup.uiDirty {
				t.Errorf("row %d moved on %s and is not marked for saving: closing Settings would lose it", row.id, k.String())
			}
		}
	}
	if checked < 20 {
		t.Errorf("only %d rows were checked; the saved groups hold more, so the selection has narrowed", checked)
	}
}

// TestOnlyAVoiceRowPreviews is 0.19.5 D-33: in Settings, `p` previews a
// voice on a voice (cast) row and on no other. A picker that is not a voice —
// the theme, history, the relay replay, every map picker — previews nothing,
// whatever list it is or is not on.
func TestOnlyAVoiceRowPreviews(t *testing.T) {
	var said []string
	d, err := NewDashboard(Config{
		Voices:         func() []string { return []string{"Samantha", "Daniel"} },
		PreviewVoice:   func(name string) { said = append(said, name) },
		VoiceInstalled: func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	d.width, d.height = 160, 60
	voices, others := 0, 0
	for _, row := range setupTable() {
		if !row.picker {
			continue
		}
		said = nil
		_, cmd := d.openSetupAt(row.id).Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
		if cmd != nil {
			cmd()
		}
		switch {
		case row.group == groupCast:
			voices++
			if len(said) != 1 {
				t.Errorf("voice row %d: p previewed %v; want its voice", row.id, said)
			}
		case len(said) != 0:
			others++
			t.Errorf("row %d (group %d) is not a voice, and p previewed %v", row.id, row.group, said)
		default:
			others++
		}
	}
	if voices < 5 || others < 15 {
		t.Errorf("checked %d voice rows and %d others; the table holds more, so the selection has narrowed", voices, others)
	}
}

// TestEveryChipWordIsRead is the reverse (0.19.5 D-36): the chip row reads
// pickerChipWords only on a row marked picker, so a word for any other row
// is never drawn, and the row's arrows go unnamed.
func TestEveryChipWordIsRead(t *testing.T) {
	for id := range pickerChipWords {
		if !setupTable()[id].picker {
			t.Errorf("pickerChipWords has %q for row %d, which is not a picker: the chip row never reads it", pickerChipWords[id], id)
		}
	}
}

// TestReduceMotionsChipsNameIt draws Settings with the focus on Reduce
// motion: the chip row names what its arrows change (0.19.5 D-36).
func TestReduceMotionsChipsNameIt(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.width, d.height = 160, 60
	got := stripANSITest(d.openSetupAt(rowReduceMotion).View().Content)
	if !strings.Contains(got, "[←→] Motion") {
		t.Error(`with the focus on Reduce motion, the chip row does not say "[←→] Motion"`)
	}
}
