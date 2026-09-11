import pathlib
# The rail stops draining first. `Next` offers a report as soon as no rail card
# is WAITING — ignoring the one still speaking — so the programme resumes
# underneath a tornado warning mid-sentence.
#
# DR-3 has said "normal programming resumes only when it is dry" since 0.14.0,
# and for two releases nothing enforced it: the Director's "is anything on the
# air anywhere" guard did, by accident, and D-82 had to remove that guard so a
# hazard could interrupt a report in the first place.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = """\t\tif _, reading := l.OnAir(t); reading {
\t\t\treturn Card{}, t, false
\t\t}
"""
assert old in s, "mN1"
p.write_text(s.replace(old, "", 1))
