import pathlib
# The offset is keyed off whether a card happens to be on the air rather than off
# the POWER. A running station between two reads — the next card still building —
# has nothing on the air for a moment, so every card shunts down a row and back
# for the length of one build. The operator reads that as the console losing its
# place, and it happens once per card for the life of the broadcast.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\tif b.power == lineup.Running {\n\t\treturn 0\n\t}\n\treturn 1"
new = "\tif _, on := b.lineup.OnAir(lineup.MainTrack); on {\n\t\treturn 0\n\t}\n\treturn 1"
assert old in s, "mP7"
p.write_text(s.replace(old, new, 1))
