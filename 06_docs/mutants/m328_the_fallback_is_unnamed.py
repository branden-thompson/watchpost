import pathlib
# 0.19.0 FR-5.2: the climatology's foF2, the fallback, is named in the words.
# Killed by TestTheFallbackIsNamed.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\twords = append(words, "foF2 from the climatology: no GloTEC grid")\n'
assert old in s, 'm328'
p.write_text(s.replace(old, '\t\twords = append(words, "")\n'))
