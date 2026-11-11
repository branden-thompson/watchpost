import pathlib
# 0.19.5 D-5 (RN-4): backspace makes the service radius the operator's own.
# Killed by TestBackspaceErasesALetterInEveryField.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\td.setup.serviceSeeded = false\n\t\td.setup.serviceMi = eraseRune(d.setup.serviceMi)\n'
assert s.count(old) == 1, 'm660'
p.write_text(s.replace(old, '\t\td.setup.serviceMi = eraseRune(d.setup.serviceMi)\n', 1))
