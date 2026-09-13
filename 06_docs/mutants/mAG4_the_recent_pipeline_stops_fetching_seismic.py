import pathlib
# The recent pipeline drops its seismic tier, so the mark goes quietly blank on
# BOTH surfaces — Observer's RECENT table and the console's pool — and neither
# looks broken (D-113).
p = pathlib.Path("app/pipelines.go"); s = p.read_text()
old = """		{Kind: snapshot.KindSeismic, Every: 15 * time.Minute}, // the regional box is shared through the client cache; RECENT half the priority cadence (seismic P2 §0.3)"""
assert old in s, "mAG4"
p.write_text(s.replace(old, "", 1))
