import pathlib
# 0.19.5 Q5.1 (DR-1): the felt bands have one owner, so moving a boundary there
# moves the screen's label and the radio's likelihood together.
# Killed by TestTheSeismicLabelAtEveryBoundary.
p = pathlib.Path('platform/render/units.go'); s = p.read_text()
old = '\tcase mag >= 4.5:\n\t\treturn 2\n'
assert s.count(old) == 1, 'm757'
p.write_text(s.replace(old, '\tcase mag >= 4.0:\n\t\treturn 2\n', 1))
