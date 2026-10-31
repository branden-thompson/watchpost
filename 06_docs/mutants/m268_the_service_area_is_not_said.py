import pathlib
# 0.19.0 FR-2.4, D-195: the words say the service area.
# Killed by TestTheServiceAreaIsSaid.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = 'var mapLayerSentences = []func(Dashboard) string{Dashboard.zoneWords, Dashboard.nightWords}'
assert old in s, 'm268'
p.write_text(s.replace(old, 'var mapLayerSentences = []func(Dashboard) string{Dashboard.nightWords}'))
