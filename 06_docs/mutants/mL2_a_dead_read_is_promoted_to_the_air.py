import pathlib
# The promotion offers a read whose context has already ended. A paused read on
# top of the suspended stack shields the dead one underneath from settle's drain
# — which stops at the first live job — so the corpse sits there and is then put
# ON THE AIR, with giveWay and resumeLine, while the chip says Paused. The
# listener presses pause and hears the previous, cancelled read resume.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "		if j := d.suspended[i]; live(j) && !j.paused {"
new = "		if j := d.suspended[i]; !j.paused {"
assert old in s, "mL2"
p.write_text(s.replace(old, new, 1))
