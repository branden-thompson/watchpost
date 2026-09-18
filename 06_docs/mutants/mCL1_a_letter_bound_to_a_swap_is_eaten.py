import pathlib
# D-148. The swap keys reach through an open text field again, so typing a place
# name loses every capital B and O — and `O` swaps the surface MID-WORD. The
# operator cannot type "Oceanside" (the station's own name) or "Bonsall" (the
# hyper-local case D-130 was ruled for) into either location field.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "	return k.Text != \"\" && r.observer.ModalOpen()"
new = "	return false"
assert old in s, "mCL1"
p.write_text(s.replace(old, new, 1))
