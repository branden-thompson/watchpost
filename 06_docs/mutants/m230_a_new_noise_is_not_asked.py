import pathlib
# 0.19.0 D-187: a snapshot for another noise is asked again.
# Killed by TestTheNoiseSettingIsTheUpdates.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '(snap.Noise == d.mapPropNoise && d.mapPane.propSnapCorrect'
assert old in s, 'm230'
p.write_text(s.replace(old, '(true && d.mapPane.propSnapCorrect'))
