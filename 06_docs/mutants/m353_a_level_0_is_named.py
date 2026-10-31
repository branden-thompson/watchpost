import pathlib
# 0.19.0 FR-1.18: only a scale above 0 is named.
# Killed by TestARaisedScaleIsNamed.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = 'func raised(level int) bool { return level >= 1 && level <= len(scaleWords) }'
assert old in s, 'm353'
p.write_text(s.replace(old, 'func raised(level int) bool { return level >= 0 && level <= len(scaleWords) }'))
