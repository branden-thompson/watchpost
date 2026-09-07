import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = 'product == "Air Quality Alert":'
assert old in s, "m42"
p.write_text(s.replace(old, 'strings.Contains(product, "Alert"):'))
