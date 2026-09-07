import pathlib
# The second attempt at the air still MOVES the card, but its effects are
# dropped: the card is on the air with no cue and no words described, so the
# band shows nothing, the voice says nothing, and the schedule waits for a
# completion for a read that was never asked for.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\td, air = d.takeTheAir()\n"
new = "\t\td, _ = d.takeTheAir()\n"
assert old in s, "mD4"
p.write_text(s.replace(old, new, 1))
