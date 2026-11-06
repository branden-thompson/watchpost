import pathlib
# 0.19.5 D-9: every word the app writes for [radio] mode still loads.
# Killed by TestTheRadioModesWordsLoad.
p = pathlib.Path('platform/config/config.go'); s = p.read_text()
old = 'radioModeSynth: true, radioModeRelay: true}'
assert s.count(old) == 1, 'm671'
p.write_text(s.replace(old, 'radioModeSynth: true}', 1))
