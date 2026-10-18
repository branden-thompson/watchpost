import pathlib
# 0.19.0 FR-5.1: the status says the stations used and the oldest reading.
# Killed by TestStationsNextAsksAgainAtOnce.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase s.Inputs.Stations > 0:\n'
assert old in s, 'm219'
p.write_text(s.replace(old, '\tcase s.Inputs.Stations > 0 && false:\n'))
