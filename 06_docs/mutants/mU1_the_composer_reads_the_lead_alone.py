import pathlib
# The Composer takes only the FIRST of the card's refs, so a burst of five reads
# one alert and calls itself done — the listener is never told about the other
# four, and nothing about the audio says so.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\t\tevs, ok := x.eventsFor(v.Refs)"
new = "\t\tevs, ok := x.eventsFor(v.Refs[:1])"
assert old in s, "mU1"
p.write_text(s.replace(old, new, 1))
