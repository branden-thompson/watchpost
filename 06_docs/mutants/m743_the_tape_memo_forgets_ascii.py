import pathlib
# 0.19.5 Q4.3 (PF-8): the tape memo keys on the glyph set, since --ascii changes the tape's marks.
# Killed by TestTheTapeKeyCoversEverythingTheTapeShows.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = 'lane: lane, ascii: o.ASCII, clock: o.Clock,'
assert s.count(old) == 1, 'm743'
p.write_text(s.replace(old, 'lane: lane, clock: o.Clock,', 1))
