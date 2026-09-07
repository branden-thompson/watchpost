import pathlib
# The state diagram taken literally: a card leaves the air only by finishing.
# A superseded or cancelled takeover becomes unexpressible, and DR-24's paired
# release has nothing to hang on.
p = pathlib.Path("platform/lineup/card.go"); s = p.read_text()
old = 'OnAir:     {name: "ON AIR", next: []State{Done, Discarded}},'
new = 'OnAir:     {name: "ON AIR", next: []State{Done}},'
assert old in s, "m62"
p.write_text(s.replace(old, new, 1))
