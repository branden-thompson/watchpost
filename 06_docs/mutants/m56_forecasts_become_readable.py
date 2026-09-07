import pathlib
p = pathlib.Path("platform/category/category.go"); s = p.read_text()
old = """			Tint: render.EventCatForecastBG, Watchlist: true},"""
new = """			Tint: render.EventCatForecastBG, Watchlist: true, ReadRank: 8},"""
assert old in s, "m56"
p.write_text(s.replace(old, new, 1))
