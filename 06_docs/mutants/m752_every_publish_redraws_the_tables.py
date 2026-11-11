import pathlib
# 0.19.5 D-31 (PF-9): a publish whose locations are the ones drawn keeps the stand-in, so the tables are not drawn again.
# Killed by TestANoChangePublishRedrawsNoTable.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = '\t\treturn sameContent{seen: next, rep: c.keyFor(old)}\n'
assert s.count(old) == 1, 'm752'
p.write_text(s.replace(old, '\t\treturn sameContent{seen: next, rep: next}\n', 1))
