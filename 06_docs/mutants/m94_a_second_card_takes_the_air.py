import pathlib
# The busy check dropped: whatever is next takes the air over whatever is already
# reading on its lane, so two voices speak at once.
#
# RE-ANCHORED AT D-82 — the guard asks about a LANE now, and it sits below `Next`
# rather than above it. m105 weakens the same line; this one deletes it. They are
# kept apart because a condition made unreachable and a condition deleted fail
# differently under a compiler, and only one of them survives `vet`.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tif _, busy := d.lineup.OnAir(track); busy {
\t\treturn d, nil, false
\t}
"""
assert old in s, "m94"
p.write_text(s.replace(old, "", 1))
