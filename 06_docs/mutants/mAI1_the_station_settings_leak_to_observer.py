import pathlib
# The station's transmitter and service radius become SHARED rows, so a listener
# with no station is offered both — the leak in the direction D-18's M4 metric
# counts, on the two settings the whole console is derived from (D-115).
p = pathlib.Path("modes/tty/setup_rows.go"); s = p.read_text()
old = """		rowTransmitter:   {rowTransmitter, groupData, scopeBroadcaster, rowInput, false, "", ""},
		rowServiceRadius: {rowServiceRadius, groupData, scopeBroadcaster, rowInput, false, "", ""},"""
assert old in s, "mAI1"
new = """		rowTransmitter:   {rowTransmitter, groupData, scopeShared, rowInput, false, "", ""},
		rowServiceRadius: {rowServiceRadius, groupData, scopeShared, rowInput, false, "", ""},"""
p.write_text(s.replace(old, new, 1))
