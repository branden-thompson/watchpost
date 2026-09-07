import pathlib
p = pathlib.Path("domains/severe/severe.go"); s = p.read_text()
old = "if r.Tab >= 0 && r.Tab < NumTabs {"
assert old in s, "m44"
p.write_text(s.replace(old, "if r.Tab >= -1 && r.Tab < NumTabs {"))
