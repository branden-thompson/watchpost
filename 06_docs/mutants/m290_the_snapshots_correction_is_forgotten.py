import pathlib
# 0.19.0 D-105: the snapshot shown keeps the correction it was asked with.
# Killed by TestTheNoReadingsCorrectionIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'd.mapPane.propSnap, d.mapPane.propSnapCorrect = &snap, v.correct'
assert old in s, 'm290'
p.write_text(s.replace(old, 'd.mapPane.propSnap, d.mapPane.propSnapCorrect = &snap, false'))
