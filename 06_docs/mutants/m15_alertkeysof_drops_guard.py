import pathlib
p = pathlib.Path("app/severe.go"); s = p.read_text()
old = """			if key, ok := severe.NormalizeID(a.ID); ok {
				keys[key] = true
			}"""
assert old in s, "m15"
p.write_text(s.replace(old, """			key, _ := severe.NormalizeID(a.ID)
			keys[key] = true"""))
