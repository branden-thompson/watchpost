# alertDuck = 0: the broadcast goes silent, which the constant's own comment forbids.
import pathlib
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = "const alertDuck = 0.15" # 0.25 until MVS-D-70
assert old in s, "m36"
p.write_text(s.replace(old, "const alertDuck = 0.0"))
