import pathlib
# D-130. The gate admits an answer for an edit the operator has already typed
# over. The resolve is in flight and cannot be called back, so the only defence
# is refusing it on arrival — without that, a slow answer about "Rainbo" lands
# under the word "Rainbow" and the window says the wrong thing about the text it
# is showing.
p = pathlib.Path("platform/debounce/debounce.go"); s = p.read_text()
old = "func (g Gate) Admits(seq int) bool { return seq == g.seq }"
new = "func (g Gate) Admits(seq int) bool { return true }"
assert old in s, "mCC3"
p.write_text(s.replace(old, new, 1))
