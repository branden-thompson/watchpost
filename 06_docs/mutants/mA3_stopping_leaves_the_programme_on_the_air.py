import pathlib
# Stopping leaves the programme on the air: the listener hears the rest of a read
# that stopped and the schedule wedges behind it.
#
# RE-ANCHORED at F-D1 round 3 (leave was split so onPowered settles once), and
# again at D-82 — where `silenceTheProgramme` stopped asking the lineup which
# LANE the on-air card was on, because it now asks the main track directly. That
# left `takeOffTheAir` as the only reader of `card`, so deleting the statement
# outright no longer compiles. The discard keeps `card` used and mutates the
# thing the rule is about: whether the card actually leaves the air.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "\td, fx, _ := d.takeOffTheAir(card.ID, Discarded) // onPowered settles once, after this\n\treturn d, fx"
new = "\t_ = card\n\treturn d, nil"
assert old in s, "mA3"
p.write_text(s.replace(old, new, 1))
