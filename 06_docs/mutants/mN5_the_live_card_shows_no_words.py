import pathlib
# The script window goes back to blanks: the LIVE card draws its headline and
# five empty rows while the station is reading from it.
#
# It is the state F-84 recorded and D-83 closed, and it is worth a mutant because
# it LOOKS FINE. A card with no words is the same shape as a card with words, by
# design (the height is fixed so nothing shifts under the operator), so the only
# thing that says the difference is what is in the rows.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\trows := b.scriptWindow(o, lane, c)\n"
new = "\trows := make([]string, bcReadLines)\n\t_, _ = o, lane\n"
assert old in s, "mN5"
p.write_text(s.replace(old, new, 1))
