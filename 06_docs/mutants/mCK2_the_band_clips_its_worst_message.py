import pathlib
# D-138. The band centres without wrapping, so a rung whose sentence is wider
# than the band is CUT — and the rung that overflows is the most severe one, the
# hazard held past its staleness bound. The console's worst message is the one
# that loses its ending.
#
# THIS IS THE DEFECT THE FIRST DRAFT SHIPPED, caught by measuring the rendered
# band rather than by reading the code.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	for _, l := range render.WrapText(plain, b.bandWidth()) { // bounded by the text (P10-02)
		body = append(body, centerText(emphasiseHeld(l, count), b.bandWidth()))
	}"""
new = """	body = append(body, centerText(emphasiseHeld(plain, count), b.bandWidth()))"""
assert old in s, "mCK2"
p.write_text(s.replace(old, new, 1))
