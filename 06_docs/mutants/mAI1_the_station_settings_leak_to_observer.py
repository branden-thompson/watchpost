import pathlib
# The station's transmitter and service radius become SHARED rows, so a listener
# with no station is offered both — the leak in the direction D-18's M4 metric
# counts, on the two settings the whole console is derived from (D-115).
#
# Re-pointed 2026-09-25 (0.18.0 D-62): the rows moved from DATA to STATION, the
# Broadcaster tab's one group.
# Killed by TestTheStationRowsAreRuledTheConsoles (0.19.5: the scope is the
# ruling's record since D-70, and an allocation count caught this only by chance).
p = pathlib.Path("modes/tty/setup_rows.go"); s = p.read_text()
old = """		rowTransmitter:   {id: rowTransmitter, group: groupStation, scope: scopeBroadcaster, kind: rowInput},
		rowServiceRadius: {id: rowServiceRadius, group: groupStation, scope: scopeBroadcaster, kind: rowInput},"""
assert old in s, "mAI1"
new = """		rowTransmitter:   {id: rowTransmitter, group: groupStation, scope: scopeShared, kind: rowInput},
		rowServiceRadius: {id: rowServiceRadius, group: groupStation, scope: scopeShared, kind: rowInput},"""
p.write_text(s.replace(old, new, 1))
