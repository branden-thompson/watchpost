import pathlib
# D-164 to D-166, FR-3.5: the words without the picture are plain.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\t\tfor _, l := range d.propBlockAs(propWordsWidth, false)[3:] {'
assert old in s, 'm166'
p.write_text(s.replace(old, '\t\tfor _, l := range d.propBlockAs(propWordsWidth, true)[3:] {'))
