import pathlib
# 0.19.5 Q8.6: accepted-costs.md's sites resolve to any top-level declaration, a constant included.
# Killed by TestAcceptedCostsNamesRealSymbols.
p = pathlib.Path('cmd/watchpost/docs_test.go'); s = p.read_text()
old = '\t\t\t\tcase *ast.ValueSpec:\n\t\t\t\t\tfor _, n := range s.Names {\n\t\t\t\t\t\tnames[n.Name] = true\n'
assert s.count(old) == 1, 'm795'
p.write_text(s.replace(old, '\t\t\t\tcase *ast.ValueSpec:\n\t\t\t\t\tfor _, n := range s.Names {\n\t\t\t\t\t\t_ = n\n', 1))
