package render

import "testing"

// ABOUT'S HOSTS READ ON THE MODAL GROUND, AND MOVE NOTHING ELSE (D-233): in
// every theme the hosts' light blue reads at AA on both modal grounds; where
// the theme's FocusCell already does, it is FocusCell's own value - the lift
// is About's alone, so the focused rows keep theirs.
func TestAboutHostReadsAndMovesNothingElse(t *testing.T) {
	was := ThemeName()
	defer SetTheme(was)
	for _, name := range ThemeNames() {
		SetTheme(name)
		tab := activeTable()
		grounds := []string{bgOf(tab, ModalBGDark), bgOf(tab, ModalBGLight)}
		if !passesAll(tab[AboutHost], grounds) {
			t.Errorf("%s: About's hosts %q do not read on the modal ground", name, tab[AboutHost])
		}
		if passesAll(tab[FocusCell], grounds) && tab[AboutHost] != tab[FocusCell] {
			t.Errorf("%s: FocusCell reads on its own, yet About's hosts are %q, not its %q", name, tab[AboutHost], tab[FocusCell])
		}
	}
}

// A CREDIT ROW'S NOTE IS A SHADE BACK, AND STILL READS (D-236): in every theme
// the note's tone reads at AA on both modal grounds, never stands out more
// than the window's text, and in the default theme sits a shade behind it.
func TestAboutNoteIsAShadeBackAndReads(t *testing.T) {
	was := ThemeName()
	defer SetTheme(was)
	for _, name := range ThemeNames() {
		SetTheme(name)
		tab := activeTable()
		ground := bgOf(tab, ModalBGDark)
		if !passesAll(tab[AboutNote], []string{ground, bgOf(tab, ModalBGLight)}) {
			t.Errorf("%s: the note's tone %q does not read on the modal ground", name, tab[AboutNote])
		}
		if Contrast(tab[AboutNote], ground) > Contrast(tab[ModalFG], ground)+0.01 {
			t.Errorf("%s: the note %q stands out more than the window's text %q", name, tab[AboutNote], tab[ModalFG])
		}
		if name == themeNames0() && Contrast(tab[AboutNote], ground) >= Contrast(tab[ModalFG], ground) {
			t.Errorf("%s: the note %q is not a shade behind the text %q", name, tab[AboutNote], tab[ModalFG])
		}
		if name == themeNames0() && Contrast(tab[AboutNote], ground) < 0.6*Contrast(tab[ModalFG], ground) {
			t.Errorf("%s: the note %q sits far behind the text %q, not a shade: %.1f:1 against %.1f:1", name, tab[AboutNote], tab[ModalFG],
				Contrast(tab[AboutNote], ground), Contrast(tab[ModalFG], ground))
		}
	}
}

// themeNames0 is the default theme's name.
func themeNames0() string { return ThemeNames()[0] }
