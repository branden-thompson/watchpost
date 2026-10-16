import pathlib
# 0.19.0 D-187: the update is asked with the Setting's noise.
# Killed by TestTheNoiseSettingIsTheUpdates.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'Noise: d.mapPropNoise, CorrectNoReadings'
assert old in s, 'm229'
p.write_text(s.replace(old, 'Noise: ionomaps.NoiseResidential, CorrectNoReadings'))
