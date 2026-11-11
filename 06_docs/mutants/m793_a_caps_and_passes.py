import pathlib
# 0.19.5 Q7.6 (D-12): a paragraph opening with an all-caps AND, BUT or SO in a changed declaration warns.
# Killed by TestTheSelfTestDiscriminates.
p = pathlib.Path('tools/authoring/caps.go'); s = p.read_text()
old = '\t\t\t\topens := n == 0 || strings.TrimSpace(g.List[n-1].Text) == "//"\n'
assert s.count(old) == 1, 'm793'
p.write_text(s.replace(old, '\t\t\t\topens := false && (n == 0 || strings.TrimSpace(g.List[n-1].Text) == "//")\n', 1))
