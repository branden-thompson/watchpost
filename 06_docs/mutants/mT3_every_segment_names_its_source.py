import pathlib
# The forecast tags EVERY one of its segments instead of the first, so a report
# of nine sentences lists itself nine times and pushes the marine, fire and
# seismic reports off the card's four manifest rows entirely.
p = pathlib.Path("domains/radio/synth/compose.go"); s = p.read_text()
old = "\t\t\tif i == 0 {\n\t\t\t\tseg.Source, seg.Detail = forecastSource, productSpan(p, now)\n\t\t\t}"
new = "\t\t\tseg.Source, seg.Detail = forecastSource, productSpan(p, now)"
assert old in s, "mT3"
p.write_text(s.replace(old, new, 1))
