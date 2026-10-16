import pathlib
# 0.19.0 D-105: the update is asked with the no-readings Setting.
# Killed by TestTheNoReadingsCorrectionIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'CorrectNoReadings: d.mapPropCorrect}'
assert old in s, 'm288'
p.write_text(s.replace(old, 'CorrectNoReadings: false}'))
