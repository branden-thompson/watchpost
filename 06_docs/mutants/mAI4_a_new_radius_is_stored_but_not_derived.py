import pathlib
# The service radius persists without re-deriving the pool, so the console names
# a service area whose candidate list belongs to the old one — the same defect
# `reStationOnCommit` exists for on the borrowed path (D-115).
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	now := lp.currentStation()
	lp.restationTo(stationArea{transmitter: now.transmitter, radiusMi: float64(mi),
		followsDefault: now.followsDefault})"""
assert old in s, "mAI4"
p.write_text(s.replace(old, "", 1))
