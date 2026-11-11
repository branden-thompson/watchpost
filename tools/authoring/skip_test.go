package main

import "testing"

// TestTheToolSkipsWhatIsNotOurs is 0.19.5 Q5.7 (DR's minor list): the tree's
// own code is read, and the vendored, the mutated, the built, the history
// and the agents' worktrees under .claude are not. A worktree is a whole
// second copy of the tree, and read it would double everything.
func TestTheToolSkipsWhatIsNotOurs(t *testing.T) {
	for _, dir := range []string{".git", ".claude", "third_party", "mutants", "dist"} {
		if !skipDir(dir) {
			t.Errorf("the tool reads %s", dir)
		}
	}
	for _, dir := range []string{"modes", "platform", "app", "domains"} {
		if skipDir(dir) {
			t.Errorf("the tool skips %s, the tree's own code", dir)
		}
	}
}
