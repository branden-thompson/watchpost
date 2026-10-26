import pathlib
# 0.19.0 R2C-2: madeWith leaves the stations option out, so it never compares equal to the Settings' options.
# Killed by TestEveryUpdateOptionIsMadeWith.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'return ionomaps.UpdateOptions{Noise: s.Noise, CorrectNoReadings: correct, NoStations: s.Inputs.NoStations}'
assert s.count(old) == 1, 'm548'
p.write_text(s.replace(old, 'return ionomaps.UpdateOptions{Noise: s.Noise, CorrectNoReadings: correct}', 1))
