import pathlib
# The drain dropped: stop returns while executors are still writing, which is
# the race that put "directory not empty" in a headless Linux run (R5-B-07).
#
# RE-ANCHORED at F-D2 round 2: stop also closes the band lane now.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\tp.jobs.Wait()\n}"
new = "}"
assert old in s, "mB6"
p.write_text(s.replace(old, new, 1))
