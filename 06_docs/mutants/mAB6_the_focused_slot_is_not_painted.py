import pathlib
# The running order's focused row loses its tint and keeps only the pointer glyph,
# so the one pointer the operator walks both tables with looks like two different
# things (D-105, HUM LEAD: "Rows in the Scheduled Line up should highlight just
# like the location pool table").
p = pathlib.Path("platform/render/lineup_table.go"); s = p.read_text()
old = """	m := tableCellStyles(data)
	if !r.Marks.Selected {
		return m
	}"""
assert old in s, "mAB6"
p.write_text(s.replace(old, """	m := tableCellStyles(data)
	if !r.Marks.Selected || true {
		return m
	}""", 1))
