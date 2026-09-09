import pathlib
# The ctrl+d window's cursor leaves the modal memo key: same defect as mV3, the
# window next door. Found while fixing that one, which is the argument for the
# rule being a table rather than a single case.
#
# THE ANCHOR MOVED AT 0.15.0, when the window became a form: what moves on the
# frame is the PICKER'S VALUE, not the question index. The mutation is unchanged
# in what it removes — everything about this window that the memo can see.
p = pathlib.Path("modes/tty/memo.go"); s = p.read_text()
old = "\t\tk.debugFocus = d.debug.focus*1000 + d.debugPick()\n"
assert old in s, "mV4"
p.write_text(s.replace(old, "", 1))
