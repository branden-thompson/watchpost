import pathlib
# RE-ANCHORED 2026-09-06: the code it patches moved out of app/ticker.go when
# that 978-line file was split along the role boundaries the architecture
# already defined (MVS-D-77/S-7). A pure move — the mutation is unchanged.
# holdRest goes back to s.hold(0) when the work outran the sound. hold(0) returns
# true WITHOUT reaching its own air check, so a takeover whose sequence ended
# during an overlapped render carries on and cues the band for a read that never
# happens — a breaking headline on the marquee with no words behind it.
#
# Reachable on every muted-class burst: the tone's duration is zero there, so the
# first remainder of the burst is already non-positive.
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	rest := d - time.Since(start)
	if rest <= 0 {"""
new = """	rest := d - time.Since(start)
	if false {"""
assert old in s, "mH0"
p.write_text(s.replace(old, new, 1))
