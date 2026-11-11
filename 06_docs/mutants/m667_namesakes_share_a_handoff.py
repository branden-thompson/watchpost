import pathlib
# 0.19.5 D-7: the hand-off cache keys on the voice's full name.
# Killed by TestAHandOffSaysTheVoicesNameWithoutItsLanguage.
p = pathlib.Path('domains/radio/synth/source.go'); s = p.read_text()
old = '\treturn "handoff:" + voiceID(from) + "\\x00" + voiceID(to) + "\\x00" + line\n'
assert s.count(old) == 1, 'm667'
p.write_text(s.replace(old, '\treturn "handoff:" + spokenName(from) + "\\x00" + spokenName(to) + "\\x00" + line\n', 1))
