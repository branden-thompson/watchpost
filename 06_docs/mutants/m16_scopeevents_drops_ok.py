import pathlib
p = pathlib.Path("app/severe.go"); s = p.read_text()
old = "			if key, ok := severe.NormalizeID(e.ID); ok && tracked[key] {"
assert old in s, "m16"
p.write_text(s.replace(old, "			if key, _ := severe.NormalizeID(e.ID); tracked[key] {"))
