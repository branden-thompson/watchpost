import pathlib
# 0.19.0 D-136, AX-8: the short names are said on opening, not after a refresh.
# Killed by TestTheShortNamesAreSaidInFull.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.mapPane.propSess.applied < 2 {\n'
assert old in s, 'm330'
p.write_text(s.replace(old, '\tif d.mapPane.propSess.applied < 99 {\n'))
