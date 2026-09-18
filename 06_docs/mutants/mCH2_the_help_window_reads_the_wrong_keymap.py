import pathlib
# D-135. The window documents the DASHBOARD's keymap again while drawing the
# console's sections, so every console action reports as unbound and the window
# comes out empty of the things it is naming. The console's bindings live on the
# Router; the Help window is Observer's, which is how the two came apart.
p = pathlib.Path("modes/tty/help_about.go"); s = p.read_text()
old = "	keys := d.helpKeys()"
new = "	keys := d.keys"
assert old in s, "mCH2"
p.write_text(s.replace(old, new, 1))
