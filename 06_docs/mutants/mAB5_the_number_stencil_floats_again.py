import pathlib
# `##.` goes back to being centred in its band, so it sits one cell right of every
# number beneath it — on both tables and at every width (D-103, HUM LEAD: "'##.'
# column head misaligned").
p = pathlib.Path("platform/render/table.go"); s = p.read_text()
old = """func stencilHeader(name string) bool { return name == "num" }"""
assert old in s, "mAB5"
p.write_text(s.replace(old, """func stencilHeader(name string) bool { return false }""", 1))
