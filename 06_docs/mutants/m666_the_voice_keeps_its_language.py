import pathlib
# 0.19.5 D-7: the identity lines say the voice's first name.
# Killed by TestAHandOffSaysTheVoicesNameWithoutItsLanguage.
p = pathlib.Path('domains/radio/synth/source.go'); s = p.read_text()
old = '\tif i := strings.Index(name, " ("); i > 0 {\n'
assert s.count(old) == 1, 'm666'
p.write_text(s.replace(old, '\tif i := strings.Index(name, " ("); i > len(name) {\n', 1))
