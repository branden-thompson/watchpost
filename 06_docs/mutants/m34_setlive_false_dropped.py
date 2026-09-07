# Drop StartSource's setLive(false): after any relay, a rendered report would
# dip to a quarter volume under an alert instead of holding, losing its words.
import pathlib
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = "	e.setLive(false) // a rendered cycle: it waits rather than plays on under an alert\n"
assert old in s, "m34"
p.write_text(s.replace(old, ""))
