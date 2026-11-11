import pathlib
# 0.19.5 Q2.13 (RN minor): Diagnostics finds the WFIGS provider by its type.
# Killed by TestDiagnosticsFindsEachFireAndQuakeSource.
p = pathlib.Path('app/stats.go'); s = p.read_text()
old = '\t\tif w, ok := pr.(*wfigs.Provider); ok {\n\t\t\tsrc.wfigs = w\n'
assert s.count(old) == 1, 'm723'
p.write_text(s.replace(old, '\t\tif w, ok := pr.(*wfigs.Provider); ok && false {\n\t\t\tsrc.wfigs = w\n', 1))
