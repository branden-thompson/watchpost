import pathlib
# The copy dropped from the reader — "it is already a fresh slice header".
# Every reader becomes a second writer into the Director's own storage.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "\tout := slices.Clone(l.tracks[t])\n\tif err := invariant.Check(len(out) == len(l.tracks[t])"
new = "\tout := l.tracks[t]\n\tif err := invariant.Check(len(out) == len(l.tracks[t])"
assert old in s, "m63"
p.write_text(s.replace(old, new, 1))
