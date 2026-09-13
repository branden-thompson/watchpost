import pathlib
# The arrival keys by the raw feed id instead of the normalised one. The two
# halves of the tie then never match: the ticker path carries the feature URL,
# the location path carries the bare OID, and the set is keyed by the bare form.
# NOTHING zone-only is ever admitted again — the over-correction, which is the
# more dangerous direction of this defect, because it silences real warnings at
# the listener's own location instead of admitting distant ones.
p = pathlib.Path("app/ticker.go"); s = p.read_text()
old = "			TrackedAs: trackedKey(e.ID),"
assert old in s, "mAP3"
p.write_text(s.replace(old, "			TrackedAs: e.ID,", 1))
