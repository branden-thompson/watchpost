import pathlib
# D-135. `[r]` no longer opens the Line-Up Request window on the console: the
# guard is never satisfied, so the key falls through to Observer, where `r` is
# the listener's repeat. The control row draws a chip for a key that does
# something else entirely.
#
# Re-pointed 2026-09-16 (D-159): the keymap switch moved into `keyAction`.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """\tcase actRequest:
\t\tif r.consoleOwnsTheKeys() {"""
new = """\tcase actRequest:
\t\tif false {"""
assert old in s, "mAZ1"
p.write_text(s.replace(old, new, 1))
