# The mirror of the fixed defect: honour the hold at open but not the dip, so a
# relay comes up at the knob's volume over a live alert for one watch tick.
import pathlib
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = "	p.SetVolume(e.volume * scale)\n	e.mu.Unlock()\n	if !hold {"
assert old in s, "m30"
p.write_text(s.replace(old, "	p.SetVolume(e.volume)\n	_ = scale\n	e.mu.Unlock()\n	if !hold {"))
