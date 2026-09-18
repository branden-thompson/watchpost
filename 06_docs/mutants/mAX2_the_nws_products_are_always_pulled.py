import pathlib
# The NWS products are fetched for every request, so a FIRE-only card pays for the
# forecast office round-trip and the UGC filtering it will not read. It is the
# most expensive of the four and the one most often not wanted.
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = "	if want.Has(report.NWS) {"
assert old in s, "mAX2"
p.write_text(s.replace(old, "	if true {", 1))
