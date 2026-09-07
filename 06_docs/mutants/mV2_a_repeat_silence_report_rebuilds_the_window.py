import pathlib
# A second silence report rebuilds the OPEN window: the listener's cursor snaps
# back to the first row and the countdown restarts at ten.
#
# The detector fires once per STREAM and the engine falls through to the next
# mount, so a station broadcasting silence on every mount — the outage this
# window exists for — reports again about every five seconds. The arrows then
# appear not to work and the clock never reaches zero, which is how it was
# reported at UAT (2026-09-05). Every test that opens the window once is blind
# to it.
p = pathlib.Path("modes/tty/relayfault.go"); s = p.read_text()
old = "\tif d.modal == modalRelayFault {\n\t\treturn d\n\t}\n"
assert old in s, "mV2"
p.write_text(s.replace(old, "", 1))
