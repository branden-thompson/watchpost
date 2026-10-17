import pathlib
# 0.19.0 AX-9: a failed answer is not available, with the key that asks again.
# Killed by TestAFailedAnswerSaysHowToAskAgain.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif d.propChartFailed() {\n\t\treturn []wordSentence{said("targets"'
assert old in s, 'm418'
p.write_text(s.replace(old, '\tif d.propChartFailed() && false {\n\t\treturn []wordSentence{said("targets"'))
