import pathlib
# 0.19.5 Q2.1 (RN-1): a muted class sounds no tone.
# Killed by TestAMutedClassSoundsNoToneAndAToneNeverDrivesTheBars.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\tif cast.Muted(class, d.tones()) {\n'
assert s.count(old) == 1, 'm675'
p.write_text(s.replace(old, '\tif false && cast.Muted(class, d.tones()) {\n', 1))
