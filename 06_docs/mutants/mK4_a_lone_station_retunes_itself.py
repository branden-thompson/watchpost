import pathlib
# A one-entry watchlist advances to itself: the wrap makes the next entry the
# current one, so the bed cuts its audio and re-tunes the same relay every dwell.
# A listener with a single station hears it restart on a timer, for no reason
# they could name.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
start = s.index("	if next == d.bed.ref {")
end = s.index("	}\n", s.index("return d, nil", start)) + 3
assert start < end, "mK4"
p.write_text(s[:start] + s[end:])
