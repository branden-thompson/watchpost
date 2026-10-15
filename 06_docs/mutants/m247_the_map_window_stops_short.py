import pathlib
# 0.19.0 D-194: the Map window reaches the bottom inset.
# Killed by TestTheMapWindowRunsInsetToInset.
p = pathlib.Path('modes/tty/view.go'); s = p.read_text()
old = 'max(5, d.height-topInset-bottomInset-2)'
assert old in s, 'm247'
p.write_text(s.replace(old, 'max(5, d.height-topInset-bottomInset-4)'))
