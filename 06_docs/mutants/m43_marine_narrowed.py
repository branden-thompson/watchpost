import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = 'case strings.Contains(product, "Marine"):'
assert old in s, "m43"
p.write_text(s.replace(old, 'case product == "Marine Weather Statement":'))
