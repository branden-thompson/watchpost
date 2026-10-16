import pathlib
# 0.19.0 D-105: a snapshot asked without the correction is asked again.
# Killed by TestTheNoReadingsCorrectionIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = ' && d.mapPane.propSnapCorrect == d.mapPropCorrect)'
assert old in s, 'm289'
p.write_text(s.replace(old, ')'))
