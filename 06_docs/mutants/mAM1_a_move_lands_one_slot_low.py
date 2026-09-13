import pathlib
# The typed SLOT is sent to `Reorder` as if it were a line-up INDEX. On a station
# at standby the two differ by one — LIVE is empty and the line-up is drawn from
# UP NEXT down (D-84) — so the card lands one place further down than the operator
# asked, silently, on the surface's normal state (D-119).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """			move(r.observer.cardID, to-r.broadcaster.liveOffset())"""
assert old in s, "mAM1"
p.write_text(s.replace(old, """			move(r.observer.cardID, to)""", 1))
