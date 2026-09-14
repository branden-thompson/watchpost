import pathlib
# `[l] Lookup Location from Pool` stops working. Drawn since D-102 and bound at
# R4b; a painted control that does nothing is worse than an absent one, because
# the operator concludes the location has nothing to show.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """		case "l":"""
assert old in s, "mAZ2"
p.write_text(s.replace(old, """		case "__unbound_l":""", 1))
