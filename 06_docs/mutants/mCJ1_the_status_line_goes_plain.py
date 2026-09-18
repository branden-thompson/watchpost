import pathlib
# D-137. The card's STATUS loses its colour coding, so "Ready for Read-Out",
# "Scheduled; Awaiting Data" and "READING" all read the same — and the one thing
# an operator checks before putting a card on the air stops being answerable at
# a glance (HUM LEAD, 2026-09-15).
p = pathlib.Path("modes/tty/broadcaster_manifest.go"); s = p.read_text()
old = '		return render.Tint("Ready for Read-Out", render.Tok(render.ProviderOK))'
new = '		return "Ready for Read-Out"'
assert old in s, "mCJ1"
p.write_text(s.replace(old, new, 1))
