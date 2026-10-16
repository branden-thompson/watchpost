import pathlib
# D-157, FR-3.5: without the picture the words give the place now.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tout = append(out, d.propOriginWords()...)\n'
assert old in s, 'm157'
p.write_text(s.replace(old, '\tout = append(out, d.propOriginWords()[:0]...)\n'))
