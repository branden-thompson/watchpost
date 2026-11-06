import pathlib
# 0.19.5 D-26 (DEF-6): Bubble Tea names the key "space", so the toggle matches that.
# Killed by TestTheRequestWindowsKeysDoWhatTheyClaim.
p = pathlib.Path('modes/tty/request.go'); s = p.read_text()
old = '\tcase "space":\n\t\t// Bubble Tea names this key "space"'
assert s.count(old) == 1, 'm657'
p.write_text(s.replace(old, '\tcase " ":\n\t\t// Bubble Tea names this key "space"', 1))
