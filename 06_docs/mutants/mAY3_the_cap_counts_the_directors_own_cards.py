import pathlib
# D-149/D-152. The overflow shed counts the RAW TRACK rather than what the
# operator can SEE, so the Director's own structural cards — not in the running
# order — count toward the cap and push real cards off the bottom.
#
# Re-pointed 2026-09-16 (P10-02): the shed loop gained an explicit bound, so the
# condition this mutates now sits in the body rather than the loop header.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = "\t\tif out.visible(t) <= MainTrackCap {"
new = "\t\tif len(out.tracks[t]) <= MainTrackCap {"
assert old in s, "mAY3"
p.write_text(s.replace(old, new, 1))
