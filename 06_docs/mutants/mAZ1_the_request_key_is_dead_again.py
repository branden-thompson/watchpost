import pathlib
# `[r]` stops opening the request window. The control row still DRAWS it, so the
# console goes on advertising a way in that does nothing — which is how it
# shipped, and what the HUM LEAD needed bound before they could UAT at all.
#
# Re-pointed 2026-09-15 (D-135): `r` is a BINDING now, not a bare key case, so
# the way to kill it is to refuse the action rather than to rename the key. The
# binding itself stays — mCH4 is the mutant that removes THAT — so this one still
# measures "the key is drawn and does nothing" and not "the key is undocumented".
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """			case actRequest:
				if r.consoleOwnsTheKeys() {"""
new = """			case actRequest:
				if false {"""
assert old in s, "mAZ1"
p.write_text(s.replace(old, new, 1))
