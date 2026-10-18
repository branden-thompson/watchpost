import pathlib
# 0.19.0 FR-1.9 (D-25, A-45): the chart answers for each continent.
# Killed by TestTheContinentsAreTheDefaultTargets.
p = pathlib.Path('platform/propanswers/answers.go'); s = p.read_text()
old = '\treturn Chart{Local: local[0], Targets: paths}, nil\n'
assert old in s, 'm190'
p.write_text(s.replace(old, '\treturn Chart{Local: local[0], Targets: nil}, nil\n'))
