import pathlib
# The enumeration offers a SUSPENDED read whose context has already ended — a
# cancelled read stays on the stack until its goroutine unwinds. The pause claims
# the corpse and reports success, so the chip says Paused while the read the
# listener can actually hear carries on.
#
# RE-ANCHORED (round 4) to the shared `live()` carrier: this deletes the
# SUSPENDED asker's use of it.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "		if j := d.suspended[i]; live(j) && j.class == narrateRead {"
new = "		if j := d.suspended[i]; j.class == narrateRead {"
assert old in s, "mK6"
p.write_text(s.replace(old, new, 1))
