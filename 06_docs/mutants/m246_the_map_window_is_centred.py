import pathlib
# 0.19.0 D-194: the Map window sits at the top inset.
# Killed by TestTheMapWindowRunsInsetToInset.
p = pathlib.Path('modes/tty/view.go'); s = p.read_text()
old = '\t\t\ty = topInset '
assert old in s, 'm246'
p.write_text(s.replace(old, '\t\t\ty = -1 '))
