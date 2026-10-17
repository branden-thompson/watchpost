import pathlib
# 0.19.0 AX-3: the hour ahead says its basis in words, not a citation.
# Killed by TestTheTypicalErrorStatesItsBasis.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = "\t\twords = append(words, b) // in words a listener can be told; the citation is the library's Source (AX-3)\n"
assert old in s, 'm399'
p.write_text(s.replace(old, '\t\twords = append(words, d.mapPane.propSnap.TypicalError.Source)\n'))
