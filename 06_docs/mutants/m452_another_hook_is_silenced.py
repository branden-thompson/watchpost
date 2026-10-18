import pathlib
# 0.19.0 I-8: make hooks refuses where another hook would stop running.
# Killed by TestHooksAreNeverSilenced.
p = pathlib.Path('tools/prepush/install.go'); s = p.read_text()
old = '\tcase cur == "" && len(active) > 0:\n'
assert old in s, 'm452'
p.write_text(s.replace(old, '\tcase false:\n'))
