import pathlib
# 0.19.0 FR-5.2: a failed update over the field kept is said first on the status line.
# Killed by TestTheStatusIsSaidWithoutThePicture.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\tif f := d.propFailedWords(); f != "" {\n'
assert old in s, 'm337'
p.write_text(s.replace(old, '\t\tif f := d.propFailedWords(); false && f != "" {\n'))
