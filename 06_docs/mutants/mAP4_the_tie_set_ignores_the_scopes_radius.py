import pathlib
# The tie set is asked for the whole world instead of the scope. severe.go says
# in as many words why that is wrong: the RECENT table is seeded with the fifty
# largest US cities and each fetches its own alerts, so an unscoped set ties a
# zone-only warning a thousand miles away, and it takes the air on a console
# scoped to twenty-five miles.
p = pathlib.Path("app/ticker.go"); s = p.read_text()
old = "	return t.severe.AlertKeysWithin(s.lat, s.lon, s.radiusMi)"
assert old in s, "mAP4"
p.write_text(s.replace(old, "	return t.severe.AlertKeysWithin(s.lat, s.lon, 100000)", 1))
