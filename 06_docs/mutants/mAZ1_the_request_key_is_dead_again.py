import pathlib
# `[r]` stops opening the request window. The control row still DRAWS it, so the
# console goes on advertising a way in that does nothing — which is how it
# shipped, and what the HUM LEAD needed bound before they could UAT at all.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """		case "r":"""
assert old in s, "mAZ1"
p.write_text(s.replace(old, """		case "__unbound_r":""", 1))
