import pathlib
# 0.19.0 FR-1.13: an hour ahead draws the climatology for that hour.
# Killed by TestTheHoursStepOnAKey.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t\tshown.Hours = []ionomaps.Hour{h}\n'
assert old in s, 'm209'
p.write_text(s.replace(old, '\t\t_ = h\n'))
