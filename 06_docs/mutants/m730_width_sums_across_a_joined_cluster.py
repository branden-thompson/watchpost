import pathlib
# 0.19.5 Q3.2 (PF-2): Width measures the stripped string whole where a cluster
# could span a sequence; summing the runs there splits the cluster.
# Killed by TestWidthMatchesStripThenMeasure.
p = pathlib.Path('platform/render/text.go'); s = p.read_text()
old = '\t\tif mayJoinAcross(s[:at], s[at+n:]) {\n'
assert s.count(old) == 1, 'm730'
p.write_text(s.replace(old, '\t\tif mayJoinAcross(s[:at], s[at+n:]) && false {\n', 1))
