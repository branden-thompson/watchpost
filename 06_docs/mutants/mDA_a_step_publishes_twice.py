import pathlib
# The split undone: stopping the radio leaves the air through leave, which
# settles and publishes, and then onPowered settles again — two publishes in one
# step with the first no longer last. The pump runs publishes concurrently, so a
# reader can apply the older snapshot after the newer one.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "\td, fx, _ := d.takeOffTheAir(card.ID, Discarded) // onPowered settles once, after this\n\treturn d, fx"
new = "\treturn d.leave(card.ID, Discarded)"
assert old in s, "mDA"
p.write_text(s.replace(old, new, 1))
