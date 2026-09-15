import pathlib
# D-135. The Help window builds Observer's sections whatever surface the
# operator is on, so a console operator who presses `?` is handed the listener's
# manual: rows for a watchlist and a visualizer the console does not have, and
# nothing about the station, the line-up or the bed (HUM LEAD, UAT 2026-09-15).
p = pathlib.Path("modes/tty/help_about.go"); s = p.read_text()
old = "	for _, g := range helpGroups(d.surface) {"
new = "	for _, g := range helpGroups(SurfaceObserver) {"
assert old in s, "mCH1"
p.write_text(s.replace(old, new, 1))
