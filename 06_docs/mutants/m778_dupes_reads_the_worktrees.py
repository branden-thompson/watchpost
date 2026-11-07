import pathlib
# 0.19.5 Q5.7: the duplicate detector skips the agents' worktrees under .claude.
# Killed by TestTheToolSkipsWhatIsNotOurs.
p = pathlib.Path('tools/dupes/main.go'); s = p.read_text()
old = '\tcase ".git", ".claude", "third_party", "mutants", "dist":\n'
assert s.count(old) == 1, 'm778'
p.write_text(s.replace(old, '\tcase ".git", "third_party", "mutants", "dist":\n', 1))
