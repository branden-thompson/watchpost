import pathlib
# D-155. Expiry is never noticed, so a hazard that lapsed hours ago stays on the
# rail: invisible to `firstStale` (which skips a zero `BuiltAt`), counted by
# `Projection`, escalated by `heldNotice` to its loudest rung telling the
# operator to go ON AIR and read it, and offered the air by `Next()` forever
# while the executor declines to build it. FR-3.3 read backwards.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\td = d.dropExpired()\n"
assert old in s, "mXP1"
p.write_text(s.replace(old, "", 1))
