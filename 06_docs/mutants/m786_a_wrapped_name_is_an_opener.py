import pathlib
# 0.19.5 Q7.1: a wrapped line that starts with the subject's name, mid-sentence, is prose.
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/rules.go'); s = p.read_text()
old = "\treturn t != \"\" && strings.ContainsAny(t[len(t)-1:], \".:!?\")\n"
assert s.count(old) == 1, 'm786'
p.write_text(s.replace(old, "\treturn t != \"\"\n", 1))
