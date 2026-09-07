import pathlib
# PL-1 undone: the pump runs each effect on its own goroutine, so the loop pays
# the 1.03 s card build and the schedule stops for it.
#
# RE-ANCHORED at F-D2 round 2: dispatch routes band work to the lane.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\t\tgo p.work(ctx, group)\n"
new = "\t\tp.work(ctx, group)\n"
assert old in s, "mB0"
p.write_text(s.replace(old, new, 1))
