import pathlib
# The location index moves back outside the memo, so it is built on every frame
# whether the tables are rebuilt or not — a map over the whole pool, allocated to
# be thrown away on every hit. Silent: the frame is correct, it just costs what
# the memo was added to stop costing.
p = pathlib.Path("modes/tty/broadcaster_memo.go"); s = p.read_text()
old = """func (b Broadcaster) tableRows(used int) []string {
	m := b.memo"""
new = """func (b Broadcaster) tableRows(used int) []string {
	_ = b.locIndex()
	m := b.memo"""
assert old in s, "mAR5"
p.write_text(s.replace(old, new, 1))
