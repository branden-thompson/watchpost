import pathlib
# The preferred send removed: both arms of the select are ready at once and Go
# picks between ready arms at random, so a run that could have been queued is
# discarded about half the time — and the run in question is the release.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """\t\tselect {
\t\tcase p.lane <- group:
\t\tdefault:
"""
new = """\t\tif false {
"""
assert old in s, "mDE"
p.write_text(s.replace(old, new, 1))
