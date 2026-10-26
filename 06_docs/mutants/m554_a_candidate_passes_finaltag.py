import pathlib
# 0.19.0 RH-9 (D-3): a paired library required at a release candidate passes a final tag.
# Killed by TestAFinalTagIsRefusedOnAReleaseCandidate.
p = pathlib.Path('tools/finaltag/main.go'); s = p.read_text()
old = 'if slices.Contains(paired, r.Path) && preRelease.MatchString(r.Version) {'
assert s.count(old) == 1, 'm554'
p.write_text(s.replace(old, 'if slices.Contains(paired, r.Path) && preRelease.MatchString(r.Path) {', 1))
