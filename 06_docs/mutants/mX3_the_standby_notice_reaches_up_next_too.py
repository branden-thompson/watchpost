import pathlib
# Every READ slot gets the standby notice instead of the LIVE slot alone, so a
# station that has just opened reports "no reports read or active" about the very
# slot the Composer is working on. D-84 puts the Composer on UP NEXT precisely so
# the line is ready at SHIFT+ENTER; the shimmer is what says that work is happening.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\t\t\tif i == 0 && b.power != lineup.Running {"
new = "\t\t\tif r.reads && b.power != lineup.Running {"
assert old in s, "mX3"
p.write_text(s.replace(old, new, 1))
