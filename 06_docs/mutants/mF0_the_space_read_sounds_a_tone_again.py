import pathlib
# MVS-D-69 undone: the [space] read opens with an attention tone again, so a
# listener who is already looking at the row is fetched by a sound — and waits
# out its two-second trailing silence before hearing a word.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = "\t\tdur := s.line(script)"
new = "\t\ts.hold(s.attention(cast.Classify(row.Product)))\n\t\tdur := s.line(script)"
assert old in s, "mF0"
p.write_text(s.replace(old, new, 1))
