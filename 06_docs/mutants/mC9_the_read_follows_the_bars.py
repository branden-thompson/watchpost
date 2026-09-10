import pathlib
# A rail card reads in the [space] read's class: the visualizer follows it and it
# queues behind a takeover instead of being one.
#
# RE-ANCHORED 2026-09-10: the class and the role are named on their own line now,
# and the mutant had stopped applying.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\tclass, role := narrateBreaking, cast.Breaking\n"
new = "\tclass, role := narrateRead, cast.Breaking\n"
assert old in s, "mC9"
p.write_text(s.replace(old, new, 1))
