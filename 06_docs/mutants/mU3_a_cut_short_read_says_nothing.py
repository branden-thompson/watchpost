import pathlib
# A read that ended early comes home silent, so the card stays ON AIR for ever
# and the band keeps a callout for a read that has stopped (DR-24).
#
# RE-ANCHORED AT THE BUILD-EXIT RED TEAM (I-2). The cut-short Failed now carries
# Routed: a read the listener stopped is a deliberate non-delivery, not the
# station going quiet, and grading it as a fault raised the RELAY FAULT window
# for an esc keypress. The mutation is unchanged — the event is deleted — and
# the corpus guard is what caught the drift.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\t\treturn []lineup.Event{lineup.Failed{ID: v.ID, Reason: "the read ended before the words did", Routed: true}}'
new = "\t\treturn nil"
assert old in s, "mU3"
p.write_text(s.replace(old, new, 1))
