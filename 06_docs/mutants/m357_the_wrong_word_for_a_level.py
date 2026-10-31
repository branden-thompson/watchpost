import pathlib
# 0.19.0 FR-1.18: a level's NOAA word.
# Killed by TestARaisedScaleIsNamed.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = '\tword := scaleWords[level-1]\n'
assert old in s, 'm357'
p.write_text(s.replace(old, '\tword := scaleWords[min(level, len(scaleWords)-1)]\n'))
