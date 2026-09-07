import pathlib
# The enumeration offers a read on the air whose context has already ended. The
# pause claims that corpse and reports success — the chip says Paused — settle
# discards it, and the live read queued behind it carries on. pauseRead says
# true while readPaused says false, at the same instant under the same lock.
#
# RE-ANCHORED (round 4): the liveness rule has one carrier, `live()`, and this
# deletes the ON-AIR asker's use of it. mK6 does the suspended clause, mL2
# innermostResumable, mL3 first().
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "	if j := d.onAir; live(j) && j.class == narrateRead {"
new = "	if j := d.onAir; j != nil && j.class == narrateRead {"
assert old in s, "mL0"
p.write_text(s.replace(old, new, 1))
