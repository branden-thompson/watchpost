import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = 'case strings.Contains(product, "Advisory"), product == "Air Quality Alert":'
assert old in s, "m37"
p.write_text(s.replace(old, 'case strings.Contains(product, "Advisory"):'))
