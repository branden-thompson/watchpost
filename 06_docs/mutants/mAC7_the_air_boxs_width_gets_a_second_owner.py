import pathlib
# `withControl` computes the air box's body width itself instead of asking the one
# owner — the shape that pushed the `b` chip off the RELAY BED row when the box
# moved into the station section and one copy of the arithmetic followed (D-107).
p = pathlib.Path("modes/tty/broadcaster_air.go"); s = p.read_text()
old = """func (b Broadcaster) withControl(line, control string) string {
	body := b.airBodyWidth()"""
assert old in s, "mAC7"
new = """func (b Broadcaster) withControl(line, control string) string {
	body := b.frameWidth() - 2 - bcAirLabelW - 1"""
p.write_text(s.replace(old, new, 1))
