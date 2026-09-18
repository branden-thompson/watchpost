import pathlib
# The helper is wrapped to a width that is not the window's — the terminal's
# instead of the 56-cell lookup window. The lines come back tinted and too long,
# the FRAME wraps them a second time, and the second wrap lands after the colour:
# "Broadcast Radius" arrives in plain grey under a red sentence.
#
# THIS IS THE SAME RULE AS mBB2 FROM THE OTHER END. mBB2 guards that the text is
# wrapped before it is tinted; this guards that it is wrapped to the width of the
# window it is going into. Wrapping to the wrong width is the same defect as not
# wrapping at all — HUM LEAD, UAT 2026-09-14, two screenshots of one rule.
p = pathlib.Path("modes/tty/modal_location.go"); s = p.read_text()
old = "poolNoteLines(o, fact, aside, modalHelperWidth(d.modalWidth()))"
new = "poolNoteLines(o, fact, aside, modalHelperWidth(o.Width))"
assert old in s, "mBB3"
p.write_text(s.replace(old, new, 1))
