import pathlib
# 0.19.5 Q2.1 (RN-1): a line that cannot be read says why on the detail line.
# Killed by TestALineThatCannotBeReadSaysWhy.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\t\td.setDetail(fmt.Sprintf("%s could not read: %v", res.Spoken, err))\n'
assert s.count(old) == 1, 'm678'
p.write_text(s.replace(old, '\t\t_ = res\n', 1))
