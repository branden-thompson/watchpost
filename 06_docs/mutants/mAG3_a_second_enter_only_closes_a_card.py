import pathlib
# The second-press rule covers the card window alone again, so `enter` on an open
# LOCATION DETAILS falls through to Observer — the same defect D-109 fixed, one
# window along (D-113).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """				if r.active == SurfaceBroadcaster &&
					(r.observer.modal == modalCard || r.observer.modal == modalDetails) {"""
assert old in s, "mAG3"
p.write_text(s.replace(old, """				if r.active == SurfaceBroadcaster && r.observer.modal == modalCard {""", 1))
