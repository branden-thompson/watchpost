import pathlib
# 0.19.0 go-ionomaps A-23: an hour ahead is labelled by the hour shown, its start.
# Killed by TestAnHourAheadIsLabelledByTheHourShown.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '.UTC().Truncate(time.Hour) // the hour shown'
assert old in s, 'm455'
p.write_text(s.replace(old, '.UTC() // the hour shown'))
