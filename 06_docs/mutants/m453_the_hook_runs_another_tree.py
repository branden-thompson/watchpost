import pathlib
# 0.19.0 H-9: the hook runs tools/prepush only from a tree that has it.
# Killed by TestTheHookRunsOnlyItsOwnTree.
p = pathlib.Path('tools/prehook/main.go'); s = p.read_text()
old = '\tif missing != nil {\n'
assert old in s, 'm453'
p.write_text(s.replace(old, '\tif missing != nil && false {\n'))
