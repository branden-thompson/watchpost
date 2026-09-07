# Drop the watchlist hint from Forecasts: the national feed cannot fill that
# tab, so a listener with no locations sees it empty forever, unexplained.
import pathlib
p = pathlib.Path("platform/category/category.go"); s = p.read_text()
old = "Tint: render.EventCatForecastBG, Watchlist: true"
assert old in s, "m46"
p.write_text(s.replace(old, "Tint: render.EventCatForecastBG"))
