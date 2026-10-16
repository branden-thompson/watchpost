import pathlib
# 0.19.0 FR-5.1: the oldest GIRO reading is the age where it is older.
# Killed by TestTheAgeIsTheOldestInputs.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif r := s.Inputs.OldestReading; s.Inputs.Stations > 0 &&'
assert old in s, 'm321'
p.write_text(s.replace(old, '\tif r := s.Inputs.OldestReading; false && s.Inputs.Stations > 0 &&'))
