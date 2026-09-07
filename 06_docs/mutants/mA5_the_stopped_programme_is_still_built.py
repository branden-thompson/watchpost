import pathlib
# The stopped programme is built anyway: a stopped radio prepares a location
# report with no cutover to be ready for, which will be stale when one comes.
#
# RE-ANCHORED at F-D1 round 2: the guard sits inside the promotion walk now.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\t\tif !d.advances(track) {
\t\t\treturn d, nil
\t\t}"""
new = """\t\tif !d.advances(track) && track == numTracks {
\t\t\treturn d, nil
\t\t}"""
assert old in s, "mA5"
p.write_text(s.replace(old, new, 1))
