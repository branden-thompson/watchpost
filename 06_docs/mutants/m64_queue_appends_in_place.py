import pathlib
# The clone dropped from Queue — "a Lineup is a value, the copy is free".
# It is not: the tracks are slices, so two lineups built from one share a
# backing array wherever Remove left capacity behind the length.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = """	out := l.clone()
	out.tracks[t] = append(out.tracks[t], c)"""
new = """	out := l
	out.tracks[t] = append(out.tracks[t], c)"""
assert old in s, "m64"
p.write_text(s.replace(old, new, 1))
