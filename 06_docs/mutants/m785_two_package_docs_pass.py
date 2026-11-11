import pathlib
# 0.19.5 Q7.1 (JM-10): a package with two package comments is reported (SN-03).
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/rules.go'); s = p.read_text()
old = '\t\tif len(ds) < 2 {\n'
assert s.count(old) == 1, 'm785'
p.write_text(s.replace(old, '\t\tif len(ds) < 3 {\n', 1))
