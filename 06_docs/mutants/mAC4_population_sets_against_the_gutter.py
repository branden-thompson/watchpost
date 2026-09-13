import pathlib
# POPULATION is set against the far side of the category gutter instead of its own
# column edge, so the digits sit three cells out from under their heading (D-106,
# HUM LEAD: "right aligned lining up with 'N' in population").
p = pathlib.Path("platform/render/table.go"); s = p.read_text()
old = """		data = append(data, padLeft(thousands(r.Population), popW))"""
assert old in s, "mAC4"
p.write_text(s.replace(old, """		data = append(data, thousands(r.Population))""", 1))
