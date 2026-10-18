import pathlib
# 0.19.0 BUILD red team round 2 (D-213, BL2-5): Settle waits for an update in flight to keep its state.
# Killed by TestAnUpdateInFlightKeepsItsStateBeforeTheEnd.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\ts.running.Add(1)\n'
assert old in s, 'm486'
p.write_text(s.replace(old, '\ts.running.Add(0)\n', 1))
