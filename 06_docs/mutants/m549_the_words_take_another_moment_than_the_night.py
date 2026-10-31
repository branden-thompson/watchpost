import pathlib
# 0.19.0 R2C-8: the words say day or night at the hour's own moment, not the night's 5-minute step, and disagree with the picture inside a step.
# Killed by TestTheWordsAndTheNightShareAMoment.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = 'd.propDayWords(origin, d.propNightMoment())'
assert s.count(old) == 1, 'm549'
p.write_text(s.replace(old, 'd.propDayWords(origin, shown.Now.At)', 1))
