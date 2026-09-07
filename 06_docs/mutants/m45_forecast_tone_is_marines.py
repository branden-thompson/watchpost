# Point Forecasts at Marine's tint: two categories rendering identically, with
# the row's colour no longer telling you which one you are looking at.
import pathlib
p = pathlib.Path("platform/category/category.go"); s = p.read_text()
old = "Tint: render.EventCatForecastBG, Watchlist: true"
assert old in s, "m45"
p.write_text(s.replace(old, "Tint: render.EventCatMarineBG, Watchlist: true"))
