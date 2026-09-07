import pathlib
# The ctrl+d window's cursor leaves the modal memo key: same defect as mV3, the
# window next door. Found while fixing that one, which is the argument for the
# rule being a table rather than a single case.
p = pathlib.Path("modes/tty/memo.go"); s = p.read_text()
old = "\t\tk.debugFocus = d.debug.focus\n"
assert old in s, "mV4"
p.write_text(s.replace(old, "", 1))
