import pathlib
# The bed stops being a resource, so a duck and a restore are dispatched to their
# own goroutines beside the read they bracket. A duck that lands after the read
# has started is the duck-lift bug in a new costume (RD-2).
#
# RE-ANCHORED AT D-82: a HAZARD's read claims the bed, a report's does not — a
# report is what the rail speaks over, not a competitor for it. The duck and the
# restore are unchanged, and they are what this removes.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tcase Duck, Restore:
\t\tout = append(out, TheBed)
"""
assert old in s, "mDC"
p.write_text(s.replace(old, "", 1))
