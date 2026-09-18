import pathlib
# D-138. The held-hazard notice loses its band and goes back to an unpainted row
# among painted regions — the least visible thing a frame can contain, on the one
# row that says a hazard is being held OFF THE AIR. HUM LEAD, 2026-09-15: "Let's
# make that look like the ticker I almost missed this."
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = '	band := o.Block(strings.Join(rows, "\\n"), render.Tok(render.AlertModalText), render.Tok(render.AlertModalAdvBG))'
# `o` IS KEPT IN USE so the mutation compiles: dropping the Block call alone
# orphans it and the build fails, which is INVALID rather than evidence.
new = '	_ = o\n	band := strings.Join(rows, "\\n")'
assert old in s, "mCK1"
p.write_text(s.replace(old, new, 1))
