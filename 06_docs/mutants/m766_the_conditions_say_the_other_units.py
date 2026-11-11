import pathlib
# 0.19.5 Q5.5 (DR-2): current conditions say the wind through windWords, in
# the listener's units.
# Killed by TestTheConditionsSayTheListenersUnits.
p = pathlib.Path('domains/radio/synth/compose.go'); s = p.read_text()
old = 'windWords(h.Wind, imperial))'
assert s.count(old) == 1, 'm766'
p.write_text(s.replace(old, 'windWords(h.Wind, !imperial))', 1))
