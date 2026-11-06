import pathlib
# 0.19.5 D-5 (RN-4): backspace in Change Position clears its error line.
# Killed by TestBackspaceErasesALetterInEveryField.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\t\tr.observer.cardMoveTo = eraseRune(r.observer.cardMoveTo)\n\t\tr.observer.cardErr = ""\n'
assert s.count(old) == 1, 'm659'
p.write_text(s.replace(old, '\t\tr.observer.cardMoveTo = eraseRune(r.observer.cardMoveTo)\n', 1))
