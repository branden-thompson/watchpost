import pathlib
# 0.19.5 Q5.6 (DR-14): an out-of-range class reads as ClassWarning in every
# accessor, through its one fallback row.
# Killed by TestToneNameNamesThePresetAndIsLoudOutOfRange.
p = pathlib.Path('domains/radio/cast/tone.go'); s = p.read_text()
old = '\tif !c.valid() {\n\t\treturn classes[ClassWarning]\n\t}\n'
assert s.count(old) == 1, 'm775'
p.write_text(s.replace(old, '\tif !c.valid() {\n\t\treturn classes[ClassDisaster]\n\t}\n', 1))
