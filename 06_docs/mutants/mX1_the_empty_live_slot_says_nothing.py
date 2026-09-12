import pathlib
# The LIVE slot goes back to a blank box on standby — the state the HUM LEAD called
# out at D-84 ("we'll need to design an empty-state for that slot") and designed at
# D-89. An operator opening the console meets an unexplained empty frame where the
# programme goes, with no way to tell "nothing is scheduled" from "the render is
# broken" — which is the same dead end "(nothing scheduled)" was replaced for.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tbody[(len(body)-1)/2] = centerText(bcStandbyNotice, l.inner())"
new = "\tbody[(len(body)-1)/2] = \"\""
assert old in s, "mX1"
p.write_text(s.replace(old, new, 1))
