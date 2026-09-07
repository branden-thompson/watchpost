import pathlib
# The lane\'s ordering guarantee removed: shared-output work is dispatched to its
# own goroutine and enqueued from there, so the ORDER it reaches the lane is
# whatever the scheduler decides rather than the order the Director described.
#
# RE-ANCHORED at F-D2 round 2.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """\t\tselect {
\t\tcase p.lane <- group:
\t\tdefault:"""
new = """\t\tgo func(group []lineup.Effect) {
\t\tselect {
\t\tcase p.lane <- group:
\t\tdefault:"""
assert old in s, "mD6 open"
s = s.replace(old, new, 1)
old2 = """\t\t\tcase <-ctx.Done():
\t\t\t\tp.jobs.Done()
\t\t\t}
\t\t}
\t}
}"""
new2 = """\t\t\tcase <-ctx.Done():
\t\t\t\tp.jobs.Done()
\t\t\t}
\t\t}
\t\t}(group)
\t}
}"""
assert old2 in s, "mD6 close"
p.write_text(s.replace(old2, new2, 1))
