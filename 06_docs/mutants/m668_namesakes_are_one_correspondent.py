import pathlib
# 0.19.5 D-7: a change between two voices sharing a first name is a change.
# Killed by TestAHandOffSaysTheVoicesNameWithoutItsLanguage.
p = pathlib.Path('domains/radio/synth/source.go'); s = p.read_text()
old = '\tchanged := s.onAir != nil && voiceID(s.onAir) != voiceID(v)\n'
assert s.count(old) == 1, 'm668'
p.write_text(s.replace(old, '\tchanged := s.onAir != nil && spokenName(s.onAir) != spokenName(v)\n', 1))
