import pathlib
# The dwell is not restarted when the advance is decided, so the deadline stays
# in the past until the new relay reports back — and every tick in between emits
# another Tune. A resolve that takes two seconds fires two more tunes behind the
# one already in flight, and the bed lands wherever the last one won.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = "	d.bed.since = d.now\n"
assert old in s, "mK0"
p.write_text(s.replace(old, "", 1))
