import pathlib
# 0.19.0 D-187: anything not offered is Residential.
# Killed by TestTheNoiseSettingIsTheUpdates.
p = pathlib.Path('modes/tty/map_prefs.go'); s = p.read_text()
old = '\treturn ionomaps.NoiseResidential\n}\n\n// propNoiseKey'
assert old in s, 'm234'
p.write_text(s.replace(old, '\treturn ionomaps.NoiseCity\n}\n\n// propNoiseKey'))
