import pathlib
# The Settings window draws every group heading whether or not this surface has
# any row under it, so the console shows "ALERTS - EVENTS" and "WATCHPOST RADIO -
# RELAY REPLAY" with nothing beneath them. A heading over no rows says a category
# of settings exists here and then shows none of it (D-92).
p = pathlib.Path("modes/tty/setup_layout.go"); s = p.read_text()
old = """		if _, drawn := visibleRowOfGroup(g, d.rowVisible); !drawn {
			continue
		}
"""
assert old in s, "mAA2"
p.write_text(s.replace(old, "", 1))
