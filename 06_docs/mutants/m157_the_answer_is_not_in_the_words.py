import pathlib
# D-157, FR-3.5: without the picture the words give the place now.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif o := d.propOriginWords(); len(o) > 0 {\n'
assert old in s, 'm157'
p.write_text(s.replace(old, '\tif o := d.propOriginWords()[:0]; len(o) > 0 {\n'))
