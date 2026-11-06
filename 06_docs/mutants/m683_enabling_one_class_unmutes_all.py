import pathlib
# 0.19.5 Q2.2 (RN-2): under an empty mute, enabling one class writes the others down first.
# Killed by TestEnablingOneClassUnderAnEmptyMuteKeepsTheOthersMuted.
p = pathlib.Path('modes/tty/setup_tones.go'); s = p.read_text()
old = '\t\tif len(d.setup.toneMuted) == 0 {\n\t\t\tfor _, k := range classRowOrder()'
assert s.count(old) == 1, 'm683'
p.write_text(s.replace(old, '\t\tif false {\n\t\t\tfor _, k := range classRowOrder()', 1))
