import pathlib
# 0.19.0 H-9: a working tree with changes the push does not carry is refused.
# Killed by TestThePushedTreeIsTheTested.
p = pathlib.Path('tools/prepush/main.go'); s = p.read_text()
old = '\tif changed != "" {\n\t\treturn "", errors.New('
assert old in s, 'm451'
p.write_text(s.replace(old, '\tif false {\n\t\treturn "", errors.New('))
