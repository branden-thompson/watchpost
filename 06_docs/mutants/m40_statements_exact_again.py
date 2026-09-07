import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = 'case strings.Contains(product, "Statement"):'
assert old in s, "m40"
p.write_text(s.replace(old, 'case product == "Special Weather Statement":'))
