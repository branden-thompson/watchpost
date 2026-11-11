import pathlib
# 0.19.5 Q9.1 (JM-10): every non-test file in modes/tty is in a row of its doc.go's map.
# Killed by TestEveryFileIsInItsPackagesMap.
p = pathlib.Path('modes/tty/doc.go'); s = p.read_text()
old = ', `poolnote.go` '
assert s.count(old) == 1, 'm796'
p.write_text(s.replace(old, '                ', 1))
