import pathlib
# 0.19.0 FR-3.7 (D-133): the words say the focus limit on opening.
# Killed by TestTheFocusLimitIsSaid.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.mapPane.propSess.applied < 2 {\n'
assert old in s, 'm329'
p.write_text(s.replace(old, '\tif d.mapPane.propSess.applied < 0 {\n'))
