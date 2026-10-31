import pathlib
# 0.19.0 D-214: the update is asked with the stations Setting.
# Killed by TestNoStationsIsASetting.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'CorrectNoReadings: d.mapPropCorrect, NoStations: d.mapPropNoGIRO}'
assert old in s, 'm394'
p.write_text(s.replace(old, 'CorrectNoReadings: d.mapPropCorrect}'))
