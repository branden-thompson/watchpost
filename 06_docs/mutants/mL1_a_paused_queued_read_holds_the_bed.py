import pathlib
# The take-back guard counts the waiting queue instead of asking first(), which
# already skips paused jobs. A read the listener paused while it waited behind an
# alert then keeps the broadcast dipped for as long as the pause lasts, with
# nobody speaking over it — the exact failure the guard's own comment forbids.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "	if d.first() == nil && d.innermostResumable() == nil {"
new = "	if len(d.waiting) == 0 && d.innermostResumable() == nil {"
assert old in s, "mL1"
p.write_text(s.replace(old, new, 1))
