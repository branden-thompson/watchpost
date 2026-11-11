import pathlib
# 0.19.5 Q7.3 (DC-9): AP-HIST-01 judges the # comments of the Makefile and the scripts.
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/rules.go'); s = p.read_text()
old = '\t\tif !strings.HasPrefix(t, "#") || (n == 0 && strings.HasPrefix(t, "#!")) || !narrates(t) {\n'
assert s.count(old) == 1, 'm787'
p.write_text(s.replace(old, '\t\tif !strings.HasPrefix(t, "#") || (n == 0 && strings.HasPrefix(t, "#!")) || narrates(t) {\n', 1))
