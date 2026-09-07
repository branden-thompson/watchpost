import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = '\tcase strings.Contains(product, "Outlook"), strings.Contains(product, "Forecast"):'
assert old in s, "m38"
i = s.index(old); j = s.index("\tcase ", i + 10)
p.write_text(s[:i] + s[j:])
