import pathlib
# Stopping leaves the programme on the air: the listener hears the rest of a
# read that stopped and the schedule wedges behind it.
#
# RE-ANCHORED at F-D1 round 3: leave was split so onPowered settles once.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "\td, fx, _ := d.takeOffTheAir(card.ID, Discarded) // onPowered settles once, after this\n\treturn d, fx"
new = "\treturn d, nil"
assert old in s, "mA3"
p.write_text(s.replace(old, new, 1))
