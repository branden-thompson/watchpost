import pathlib
# first() skips a listener's hold but not a job whose context has ended, so the
# take-back guard keeps the broadcast dipped for a read nobody is going to hear
# — until that read's own goroutine gets around to unwinding.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "		if j.paused || !live(j) {"
new = "		if j.paused {"
assert old in s, "mL3"
p.write_text(s.replace(old, new, 1))
