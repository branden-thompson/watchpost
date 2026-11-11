import pathlib
# 0.19.5 Q5.5 (DR-2): a depth is kilometres whatever the listener's units.
# Killed by TestADepthIsSaidInWholeKilometres.
p = pathlib.Path('domains/radio/synth/spoken_units.go'); s = p.read_text()
old = 'return distanceWords(km, false) }'
assert s.count(old) == 1, 'm767'
p.write_text(s.replace(old, 'return distanceWords(km, true) }', 1))
