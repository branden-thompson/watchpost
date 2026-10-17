import pathlib
# 0.19.0 AX-8: the short names are said in full on opening.
# Killed by TestTheShortNamesAreSaidInFull.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = 'said("focus", propFocusLimit), said("terms", propTerms))'
assert old in s, 'm412'
p.write_text(s.replace(old, 'said("focus", propFocusLimit))'))
