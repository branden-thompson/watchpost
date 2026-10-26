import pathlib
# 0.19.0 RC-9: a tagged test counts as selected by a recipe that never builds it, so a gate whose -tags was dropped passes on zero tests.
# Killed by TestTheRunSelectorGuardCatchesAnEmptySelection.
p = pathlib.Path('cmd/watchpost/gates_test.go'); s = p.read_text()
old = '\t\t\t\tif re.MatchString(n.name) && sel.builds(n) {\n'
assert s.count(old) == 1, 'm557'
p.write_text(s.replace(old, '\t\t\t\tif re.MatchString(n.name) && (sel.builds(n) || true) {\n', 1))
