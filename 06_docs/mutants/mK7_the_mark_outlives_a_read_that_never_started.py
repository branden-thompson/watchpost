import pathlib
# The mark is cleared only from inside the narration sequence, which a read whose
# ROW HAS GONE never reaches — the row can vanish from the feed between the frame
# the listener saw and the key they pressed. The window keeps a play mark for a
# read that never began, and nothing can take it down.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = """		r.clearMarkUnlessReplaced(gen)
		close(done)"""
new = """		close(done)"""
assert old in s, "mK7"
p.write_text(s.replace(old, new, 1))
