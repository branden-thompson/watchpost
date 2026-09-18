package gateoracle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneForOracle lays out the tree AS CI HAS IT under root/tree: a shared clone
// of source at its commit, HEAD detached, no tags, with source's tracked and
// unignored files over it — never a git-ignored file, never dist/.
func CloneForOracle(t Reporter, source, root string) string {
	t.Helper()
	tree := filepath.Join(root, "tree")
	git := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("COULD NOT RUN — git %v in %s: %v\n%s", args, dir, err, out)
		}
		return string(out)
	}
	git(source, "clone", "--shared", "--quiet", source, tree)
	git(tree, "checkout", "--quiet", "--detach")
	for _, tag := range strings.Fields(git(tree, "tag", "--list")) { // bounded by the tag list (P10-02)
		git(tree, "tag", "-d", tag)
	}
	listed := git(source, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	for _, rel := range strings.Split(listed, "\x00") { // bounded by the file list (P10-02)
		if rel == "" || strings.HasPrefix(rel, "dist/") {
			continue
		}
		copyEntry(t, filepath.Join(source, rel), filepath.Join(tree, rel), rel)
	}
	// THE WORKING TREE IS COMMITTED IN THE SCRATCH, so a predicate on cleanliness
	// answers as it does in CI whatever the developer has uncommitted, and every
	// run can reset to this commit.
	git(tree, "add", "-A")
	if status := git(tree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		git(tree, "-c", "user.name=oracle", "-c", "user.email=oracle", "commit", "-q", "--no-verify", "-m", "oracle: the working tree")
	}
	return tree
}

// copyEntry mirrors one working-tree entry: a file with its mode, a symlink as a
// symlink, and a tracked file deleted in the working tree as absent.
func copyEntry(t Reporter, src, dst, rel string) {
	info, err := os.Lstat(src)
	if err != nil {
		_ = os.Remove(dst)
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	_ = os.Remove(dst)
	if info.Mode()&os.ModeSymlink != 0 {
		link, _ := os.Readlink(src)
		_ = os.Symlink(link, dst)
		return
	}
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("COULD NOT RUN — copying %s: %v", rel, err)
	}
	if err := os.WriteFile(dst, body, info.Mode().Perm()); err != nil {
		t.Fatalf("COULD NOT RUN — writing %s: %v", rel, err)
	}
}
