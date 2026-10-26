import pathlib
# 0.19.0 R2H-3: a replace of a paired library passes a final tag.
# Killed by TestAFinalTagIsRefusedOnAReleaseCandidate.
p = pathlib.Path('tools/finaltag/main.go'); s = p.read_text()
old = '\t\tif slices.Contains(paired, r.Old.Path) {\n'
assert s.count(old) == 1, 'm553'
p.write_text(s.replace(old, '\t\tif slices.Contains(paired, r.Old.Path) && r.New.Version == "" && false {\n', 1))
