import pathlib
# 0.19.0 D-127: a target's open bands before its closed.
# Killed by TestTheBandsAreSaidOpenFirst.
p = pathlib.Path('platform/propanswers/words.go'); s = p.read_text()
old = '\t{ionomaps.Open, "open on "},\n\t{ionomaps.Disturbed, "disturbed on "},\n\t{ionomaps.Absorbed, "absorbed on "},\n\t{ionomaps.Weak, "too weak on "},\n\t{ionomaps.AboveUpper, "above the upper limit on "},\n'
assert old in s, 'm341'
p.write_text(s.replace(old, '\t{ionomaps.AboveUpper, "above the upper limit on "},\n\t{ionomaps.Absorbed, "absorbed on "},\n\t{ionomaps.Weak, "too weak on "},\n\t{ionomaps.Open, "open on "},\n\t{ionomaps.Disturbed, "disturbed on "},\n'))
