import pathlib
# The RELAY REPLAY row is drawn perfectly and does nothing.
#
# setup.go gates ←→ on the row's `picker` field, whose NAME says "carries a voice
# picker" and whose JOB is "the arrow keys cycle this". A new picker row that
# leaves it false renders the label, the value and both chips, and ignores every
# press. This was the real state of the row when it was first written, and three
# tests passed over it: they called cycleRelayDwell directly and never touched a
# key.
#
# RE-ANCHORED 2026-09-12 (D-92): the row gained a `scope` field between its group
# and its kind.  The rule it guards is unchanged.
p = pathlib.Path("modes/tty/setup_rows.go"); s = p.read_text()
old = "rowRelayDwell: {rowRelayDwell, groupRelay, scopeObserver, rowPicker, true,"
new = "rowRelayDwell: {rowRelayDwell, groupRelay, scopeObserver, rowPicker, false,"
assert old in s, "mM4"
p.write_text(s.replace(old, new, 1))
