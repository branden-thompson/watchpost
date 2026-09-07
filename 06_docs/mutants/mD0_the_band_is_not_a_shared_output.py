import pathlib
# The band stops being a resource, so a cue and a release ride their own
# goroutines. The release of the card LEAVING the air and the cue of the card
# TAKING it then race — and when the release lands second it clears the callout
# the cue just put up, so the next card reads with the band already back on
# rotation.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tcase CueTicker, ReleaseTicker:
\t\tout = append(out, TheBand)
"""
assert old in s, "mD0"
p.write_text(s.replace(old, "", 1))
