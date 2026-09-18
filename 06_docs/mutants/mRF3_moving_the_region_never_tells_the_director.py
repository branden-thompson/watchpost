import pathlib
# D-154. The station's region moves and the Director is never told, which is the
# defect as it actually shipped: `d.settings.Fence` had ONE assignment in
# `platform/lineup`, behind the air-moved guard, so narrowing the service radius
# in Settings had no path to the rail at all. The handler exists and nothing
# calls it — a fix that is present in mechanism and absent in wiring.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """	if mc := lp.masterControl(); mc != nil {
		mc.Refence()
	}
"""
assert old in s, "mRF3"
p.write_text(s.replace(old, "", 1))
