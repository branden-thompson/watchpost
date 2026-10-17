import pathlib
# 0.19.0 FR-3.1 (D-127): one line per target, its name first.
# Killed by TestThePropagationWordsAnswerWithoutThePicture.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tout = append(out, wordSentence{keys: keys, text: spoken(listWords(named[say])+": "+strings.TrimPrefix(say, "\\x00")) + "."})\n'
assert old in s, 'm347'
p.write_text(s.replace(old, '\t\tout = append(out, wordSentence{keys: keys, text: spoken(strings.TrimPrefix(say, "\\x00")) + "."})\n'))
