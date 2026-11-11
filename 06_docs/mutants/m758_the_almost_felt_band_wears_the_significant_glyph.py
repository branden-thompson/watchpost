import pathlib
# 0.19.5 Q5.1 (DR-1): the two felt bands share the felt glyph; only the
# significant band wears the significant one.
# Killed by TestTheSeismicLevelAtEveryBoundary.
p = pathlib.Path('platform/render/units.go'); s = p.read_text()
old = 'var seismicLevels = [...]int{1, 2, 2, 3}'
assert s.count(old) == 1, 'm758'
p.write_text(s.replace(old, 'var seismicLevels = [...]int{1, 2, 3, 3}', 1))
