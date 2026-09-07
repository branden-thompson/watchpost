import pathlib
# The bed dips to a level a listener called distracting: loud enough to compete
# with the alert being read over it.
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = "const alertDuck = 0.15"
new = "const alertDuck = 0.85"
assert old in s, "mF1"
p.write_text(s.replace(old, new, 1))
