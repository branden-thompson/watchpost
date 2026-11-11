import pathlib
# 0.19.5 Q7.6 (D-11): a release named earlier in a sentence qualifies the IDs after it.
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/cite.go'); s = p.read_text()
old = "\t\tcase m.kind == 'r':\n\t\t\tqualified = true\n"
assert s.count(old) == 1, 'm791'
p.write_text(s.replace(old, "\t\tcase m.kind == 'r':\n\t\t\tqualified = false\n", 1))
