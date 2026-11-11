import pathlib
# 0.19.5 Q5.6 (DR-8): a spoken station name is bounded at 48 runes.
# Killed by TestAStationNameIsBoundedInRunes.
p = pathlib.Path('domains/radio/synth/marine.go'); s = p.read_text()
old = 'const maxStationNameRunes = 48'
assert s.count(old) == 1, 'm769'
p.write_text(s.replace(old, 'const maxStationNameRunes = 50', 1))
