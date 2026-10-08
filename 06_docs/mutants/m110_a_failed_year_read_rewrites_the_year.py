import pathlib
# #27 (0.19.0 FR-6.5, W1.1, W1.2): a roll-up part that exists but cannot be read is never written over.
# Killed by TestAFailedYearReadNeverRewritesTheYear.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "		doc, ok := readAs[yearDoc](s, path, d)\n		if !ok {\n			return \"\", yearDoc{}, false\n		}"
assert old in s, "m110"
p.write_text(s.replace(old, "		doc, ok := readAs[yearDoc](s, path, d)\n		if !ok && false {\n			return \"\", yearDoc{}, false\n		}"))
