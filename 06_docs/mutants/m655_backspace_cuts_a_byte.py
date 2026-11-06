import pathlib
# 0.19.5 D-5 (DEF-1): backspace takes the last letter whole.
# Killed by TestBackspaceErasesALetterInEveryField.
p = pathlib.Path('modes/tty/textfield.go'); s = p.read_text()
old = '\treturn s[:len(s)-size]\n'
assert s.count(old) == 1, 'm655'
p.write_text(s.replace(old, '\treturn s[:len(s)-min(size, 1)]\n', 1))
