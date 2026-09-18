import pathlib
# The tracked tie dropped ENTIRELY: every zone-only alert reaches a scoped
# surface, so the burst and the tape disagree about one hazard.
#
# Re-pointed at D-122, which moved the tie from the arrival to the fence. This
# is the "admit everything" half; mAP1 is the subtler one that kept the
# arrival's own frozen claim, which is the defect that actually shipped.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "		return f.tracks(a.TrackedAs)"
new = "		return true"
assert old in s, "m85"
p.write_text(s.replace(old, new, 1))
