import pathlib
# 0.19.5 Q7.2 (DC-12): a ratified duplicate row that matches no group is named.
# Killed by TestAStaleRowIsNamed.
p = pathlib.Path('tools/dupes/main.go'); s = p.read_text()
old = '\t\tif !found[fp] {\n\t\t\tstale = append(stale, fp)\n'
assert s.count(old) == 1, 'm783'
p.write_text(s.replace(old, '\t\tif !found[fp] && false {\n\t\t\tstale = append(stale, fp)\n', 1))
