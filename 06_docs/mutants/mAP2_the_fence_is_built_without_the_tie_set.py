import pathlib
# The rule is right and nobody hands it the data. `Fence.Admits` can only ask a
# set it was given, so a fence built without one follows nothing and every
# zone-only alert is fenced out — including the flood warning at the listener's
# own watched location. The platform tests stay green: they build the fence
# themselves. Only a test that drives `deck.fence()` sees it (P-1).
p = pathlib.Path("app/ticker.go"); s = p.read_text()
old = """	return lineup.Fence{RadiusMi: s.radiusMi, Lat: s.lat, Lon: s.lon, HasOrigin: true,
		Tracked: t.tiesWithin(s)}"""
assert old in s, "mAP2"
p.write_text(s.replace(old, "	return lineup.Fence{RadiusMi: s.radiusMi, Lat: s.lat, Lon: s.lon, HasOrigin: true}", 1))
