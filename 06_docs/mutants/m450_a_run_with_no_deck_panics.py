import pathlib
# 0.19.0 C-6: attaching no deck tells it nothing.
# Killed by TestARunWithNoDeckStarts.
p = pathlib.Path('app/dashboard.go'); s = p.read_text()
old = '\tif deck == nil {\n\t\treturn // no deck could be built (attachRadio): nothing to tell it\n\t}\n'
assert old in s, 'm450'
p.write_text(s.replace(old, ''))
