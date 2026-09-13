import pathlib
# The bed gets its own labelled row back in the station lines, beside the row it
# already has in the air box — two selectors and two state words for one bed
# (D-107, HUM LEAD: "the BED is now duplicated in the UI - which is confusing").
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	if b.statusNote != "" {
		rows = append(rows, render.TruncateCells(label("")+why, max(0, lane)))
	}
	return rows"""
assert old in s, "mAC5"
new = """	if b.statusNote != "" {
		rows = append(rows, render.TruncateCells(label("")+why, max(0, lane)))
	}
	rows = append(rows, render.PadTo(label(o.KeyCap("b")+" BED:")+b.bedSelector(o), lane))
	return rows"""
p.write_text(s.replace(old, new, 1))
