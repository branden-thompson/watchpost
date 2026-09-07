import pathlib
# A read a listener paused while it was SUSPENDED is counted as holding the bed,
# so the broadcast stays dipped for as long as the pause lasts — with nobody
# speaking over it. The listener presses pause expecting the radio back and gets
# a station that has gone quiet instead.
#
# RE-ANCHORED (round 3): the take-back guard now asks first() rather than
# counting the queue, and mL1 carries that half. This one deletes the SUSPENDED
# half — the filter that excludes a paused job from the resumable set.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "	if d.first() == nil && d.innermostResumable() == nil {"
new = "	if d.first() == nil && len(d.suspended) == 0 {"
assert old in s, "mJ0"
p.write_text(s.replace(old, new, 1))
