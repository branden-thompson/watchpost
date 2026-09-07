import pathlib
# The precedence reversed — the tidy-up that reads as "tracks in declaration
# order". An alert waits behind the watchlist rotation.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "for _, t := range []Track{AlertRail, MainTrack} {"
new = "for _, t := range []Track{MainTrack, AlertRail} {"
assert old in s, "m58"
p.write_text(s.replace(old, new, 1))
