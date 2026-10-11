import pathlib
# D-170: a row's sources are drawn as their chips.
# Killed by TestTheSourcesAreTheirBadges.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\t\t\tl = strings.ReplaceAll(l, "  "+b+"  ", chipFace(b))\n'
assert old in s, 'm186'
p.write_text(s.replace(old, '\t\t\tl = strings.ReplaceAll(l, "  "+b+"  ", "  "+b+"  ")\n'))
