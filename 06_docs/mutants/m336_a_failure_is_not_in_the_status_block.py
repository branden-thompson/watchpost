import pathlib
# 0.19.0 FR-5.2: a failed update over the field kept is said in the words.
# Killed by TestTheStatusIsSaidWithoutThePicture.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif w := d.propFailedWords(); w != "" {\n'
assert old in s, 'm336'
p.write_text(s.replace(old, '\tif w := d.propFailedWords(); false && w != "" {\n'))
