import pathlib
# D-157, FR-3.5: without the picture the words give the place now.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\t\tfor _, l := range d.propBlock(propWordsWidth)[3:] {'
assert old in s, 'm157'
p.write_text(s.replace(old, '\t\tfor _, l := range d.propBlock(propWordsWidth)[6:] {'))
