import pathlib
# 0.19.0 D-197: every chart row names its best frequency.
# Killed by TestTheBestFrequencyIsNamed.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = '\treturn limits + " · " + best\n'
assert old in s, 'm274'
p.write_text(s.replace(old, '\treturn limits\n'))
