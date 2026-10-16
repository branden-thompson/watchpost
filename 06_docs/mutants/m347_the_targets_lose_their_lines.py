import pathlib
# 0.19.0 FR-3.1 (D-127): one line per target, its name first.
# Killed by TestThePropagationWordsAnswerWithoutThePicture.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tline := names[i] + ": " + propBandsSaid(a.Bands)\n'
assert old in s, 'm347'
p.write_text(s.replace(old, '\t\tline := names[i][:0] + propBandsSaid(a.Bands)\n'))
