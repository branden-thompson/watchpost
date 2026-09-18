import pathlib
# The relay selector goes back to PUBLISHING its choice and keeping nothing, which
# is the original defect one layer down: the message reaches the console and the
# next settle has no way to know what was chosen, so it publishes the Director's
# bed over it. A fact that is announced but not owned has no owner at all.
p = pathlib.Path("app/bedrelay.go"); s = p.read_text()
old = "\tlp.bedRelay = relayLine(chosen)\n\tline := lp.bedRelay"
new = "\tline := relayLine(chosen)"
assert old in s, "mY2"
p.write_text(s.replace(old, new, 1))
