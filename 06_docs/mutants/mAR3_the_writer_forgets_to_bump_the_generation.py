import pathlib
# The generation stops moving with the value it stands for. This is the OTHER
# half of the same rule and the easier one to break, because the assignment still
# looks right: `b.lineup = v.Lineup` is exactly what it always was, and the
# counter beside it is the only thing that tells the memo anything happened.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "		b.lineup, b.lineupGen = v.Lineup, b.lineupGen+1"
assert old in s, "mAR3"
p.write_text(s.replace(old, "		b.lineup = v.Lineup", 1))
