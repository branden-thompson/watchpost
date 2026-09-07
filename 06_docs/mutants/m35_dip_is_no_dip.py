# alertDuck = 1: a relay plays at the listener's full volume over a live alert.
import pathlib
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = "const alertDuck = 0.15" # 0.25 until MVS-D-70
assert old in s, "m35"
p.write_text(s.replace(old, "const alertDuck = 1.0"))
