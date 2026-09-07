# Let laneRowsPerTab sort the caller's own rows instead of its per-tab copies:
# publish reuses that slice for the window, so the window would silently start
# listing by severity while the tape lists by recency.
import pathlib
p = pathlib.Path("app/severe.go"); s = p.read_text()
old = "func laneRowsPerTab(rows []severe.Row) []severe.Row {"
assert old in s, "m33"
p.write_text(s.replace(old, old + chr(10) + chr(9) + "sort.SliceStable(rows, func(i, j int) bool { return rows[i].Severity > rows[j].Severity })"))
