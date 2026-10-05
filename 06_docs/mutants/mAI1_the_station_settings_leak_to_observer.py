import pathlib
# The station's transmitter and service radius become SHARED rows, so a listener
# with no station is offered both — the leak in the direction D-18's M4 metric
# counts, on the two settings the whole console is derived from (D-115).
#
# Re-pointed 2026-09-25 (0.18.0 D-62): the rows moved from DATA to STATION, the
# Broadcaster tab's one group.
p = pathlib.Path("modes/tty/setup_rows.go"); s = p.read_text()
old = """		rowTransmitter:   {rowTransmitter, groupStation, scopeBroadcaster, rowInput, false, "", ""},
		rowServiceRadius: {rowServiceRadius, groupStation, scopeBroadcaster, rowInput, false, "", ""},"""
assert old in s, "mAI1"
new = """		rowTransmitter:   {rowTransmitter, groupStation, scopeShared, rowInput, false, "", ""},
		rowServiceRadius: {rowServiceRadius, groupStation, scopeShared, rowInput, false, "", ""},"""
p.write_text(s.replace(old, new, 1))
