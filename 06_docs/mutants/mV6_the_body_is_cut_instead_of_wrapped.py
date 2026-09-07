import pathlib
# A body line too long for the window is CUT rather than wrapped, which is the
# class UAT 25 ruled out on WrapLines in as many words. In this window what gets
# cut is the address of the station the listener is being told to tune to, in
# the one window that exists because something is already wrong.
#
# RE-ANCHORED at the T3.10 red team: the inset-and-wrap is one shared helper now
# (insetModalLines), so the rule is guarded once for every window that uses it
# rather than once per window.
p = pathlib.Path("modes/tty/columns.go"); s = p.read_text()
old = "\t\tfor _, w := range render.WrapLines([]string{strings.TrimLeft(l, \" \")}, content) {\n\t\t\tout = append(out, pad+w)\n\t\t}"
new = "\t\tout = append(out, pad+render.TruncateCells(strings.TrimLeft(l, \" \"), content))"
assert old in s, "mV6"
p.write_text(s.replace(old, new, 1))
