import pathlib
# The card's title stops naming what KIND of read it is, so every card in the
# running order reads as a bare place name — which is what the HUM LEAD reported
# in UAT: "Oceanside, CA" where the reference draws "LOCATION REPORT • Oceanside,
# CA". The kind is the slot registry's and the subject is the producer's; this
# drops the half the console owns.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\theadline := kindFirst(cardTitle(c, l.g), l.lane, l.g)"
new = "\theadline := kindFirst(plaintext.Text(c.Headline), l.lane, l.g)"
assert old in s, "mT4"
p.write_text(s.replace(old, new, 1))
