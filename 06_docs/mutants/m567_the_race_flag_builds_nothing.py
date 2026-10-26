import pathlib
# 0.19.0 RC-9 (R3C-12): the run-selector check ignores -race, so a race-only test counts as unselected under a recipe that runs it.
# Killed by TestTheRunSelectorGuardCatchesAnEmptySelection.
p = pathlib.Path('cmd/watchpost/gates_test.go'); s = p.read_text()
old = '(tag == "race" && sel.race)'
assert s.count(old) == 1, 'm567'
p.write_text(s.replace(old, '(tag == "race" && sel.race && false)', 1))
