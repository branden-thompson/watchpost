import pathlib
# D-158. The Router answers the operator's rebound chord and the HELP still
# prints the default — the one surface whose entire job is to say which key to
# press, telling them the wrong one. Worse than no help: it is confidently wrong
# about the thing it exists for.
p = pathlib.Path("modes/tty/help_about.go"); s = p.read_text()
old = "\tbc := d.consoleKeyMap()"
assert old in s, "mKB2"
p.write_text(s.replace(old, "\tbc := broadcasterKeyMap()", 1))
