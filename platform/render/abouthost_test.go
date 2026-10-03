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
