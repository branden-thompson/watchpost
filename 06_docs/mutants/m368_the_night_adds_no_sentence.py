import pathlib
# 0.19.0 FR-3.3 (D-199): the night is a layer that adds its sentence.
# Killed by TestALayerAddsItsSentence.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = 'var mapLayerSentences = []func(Dashboard) string{Dashboard.zoneWords, Dashboard.nightWords}'
assert old in s, 'm368'
p.write_text(s.replace(old, 'var mapLayerSentences = []func(Dashboard) string{Dashboard.zoneWords}'))
