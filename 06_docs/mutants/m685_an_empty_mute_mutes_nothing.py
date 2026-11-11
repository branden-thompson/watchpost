import pathlib
# 0.19.5 Q2.2 (RN-2): under mute mode an empty set means every class.
# Killed by TestEnablingOneClassUnderAnEmptyMuteKeepsTheOthersMuted.
p = pathlib.Path('modes/tty/setup_tones.go'); s = p.read_text()
old = '\tif len(st.toneMuted) == 0 {\n\t\treturn true\n'
assert s.count(old) == 1, 'm685'
p.write_text(s.replace(old, '\tif len(st.toneMuted) == 0 {\n\t\treturn false\n', 1))
