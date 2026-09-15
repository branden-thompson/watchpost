import pathlib
# D-135. `[r]` goes back to being a bare key case instead of a binding, so it is
# neither rebindable nor documentable — D-15 ("keys are data") broken in both
# directions the rule exists for. The control row still draws `[r]` and the Help
# window still cannot name it.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = '		actRequest:       {Keys: []string{"r"}, Help: "Line-Up Request"},\n'
assert old in s, "mCH4"
p.write_text(s.replace(old, "", 1))
