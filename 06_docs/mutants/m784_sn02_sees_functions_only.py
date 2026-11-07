import pathlib
# 0.19.5 Q7.1 (DC-1, JR-2): SN-02 judges types, values and fields, not functions alone.
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/rules.go'); s = p.read_text()
old = '\t\t\t\tadd(specNames(sp), specDoc(sp, x))\n'
assert s.count(old) == 1, 'm784'
p.write_text(s.replace(old, '\t\t\t\t_ = sp\n', 1))
