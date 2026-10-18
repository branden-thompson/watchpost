import pathlib
# 0.19.0 BUILD red team round 2 (FR-3.1, PN2-10): an hour ahead not yet worked out is said so, never with now's figures.
# Killed by TestAnHourAheadIsNotSaidBeforeItIsWorkedOut.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif !ready && d.mapPane.propAhead > 0 {'
assert old in s, 'm479'
p.write_text(s.replace(old, '\tif !ready && d.mapPane.propAhead > 99 {', 1))
