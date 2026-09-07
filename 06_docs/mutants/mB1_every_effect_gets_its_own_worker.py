import pathlib
# The grouping undone: a card\'s effects no longer share a run, so a cue and its
# words end up on separate workers — racing each other, with the band promising
# a callout after the read has started.
#
# RE-ANCHORED at F-D2 round 2: grouping is runsOf, connected by card.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """\t\t\tif i, seen := of[id]; seen {
\t\t\t\truns[i] = append(runs[i], f)
\t\t\t\tcontinue
\t\t\t}
"""
assert old in s, "mB1"
p.write_text(s.replace(old, "", 1))
