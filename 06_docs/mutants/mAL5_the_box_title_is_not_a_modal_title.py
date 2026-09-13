import pathlib
# The box's title goes back to the ground's base grey, so the one titled thing on
# the frame does not read like a title — on a box that now wears the modal's own
# tile (D-114, D-118).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = '\thead := strings.Repeat(mark, 3) + " " + render.Tint(title, render.Tok(render.ModalTitle)) + " "'
new = '\thead := strings.Repeat(mark, 3) + " " + title + " "'
assert old in s, "mAL5"
p.write_text(s.replace(old, new, 1))
