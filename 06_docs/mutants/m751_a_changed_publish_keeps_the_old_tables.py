import pathlib
# 0.19.5 D-31 (PF-9): a publish whose locations changed stands for itself, so the tables are drawn again.
# Killed by TestAChangedPublishRedrawsTheTables.
p = pathlib.Path('modes/tty/memo.go'); s = p.read_text()
old = '\tif old != nil && next != nil && reflect.DeepEqual(old.Locations, next.Locations) {\n'
assert s.count(old) == 1, 'm751'
p.write_text(s.replace(old, '\tif old != nil && next != nil && (reflect.DeepEqual(old.Locations, next.Locations) || true) {\n', 1))
