import pathlib
# The divert count taken from the raw arrivals instead of what the burst could
# have read. A forecast the listener never lost is spoken as an alert they did.
#
# RE-ANCHORED at the T3.10 red team: `cards` is gone — the selection IS the
# takeover's refs now, and the count is taken from the chosen placements.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = "\tdivert := len(cand) - len(chosen)"
new = "\tdivert := len(arrivals) - len(chosen)"
assert old in s, "m72"
p.write_text(s.replace(old, new, 1))
