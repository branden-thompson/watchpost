import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = 'case strings.Contains(product, "Outlook"), strings.Contains(product, "Forecast"):'
assert old in s, "m41"
p.write_text(s.replace(old, 'case strings.Contains(product, "Outlook"):'))
