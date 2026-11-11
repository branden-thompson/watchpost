import pathlib
# 0.19.5 Q8.5 (D-15): the atlas draws docs/ and the as-built maps, not a plan's or a report's diagram.
# Killed by TestOnlyTheAsBuiltDiagramsAreDrawn.
p = pathlib.Path('tools/atlas/main.go'); s = p.read_text()
old = '\treturn root == "docs" || strings.HasPrefix(filepath.Base(path), "as-built")\n'
assert s.count(old) == 1, 'm794'
p.write_text(s.replace(old, '\treturn root != ""\n', 1))
