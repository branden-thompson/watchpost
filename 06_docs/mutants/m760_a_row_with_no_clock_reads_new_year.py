import pathlib
# 0.19.5 Q5.2 (DR-5): a zero time reads blank, never 01/01 00:00 (R3-A-06).
# Killed by TestTheSevereRowsTimesReadInThePlacesZone.
p = pathlib.Path('domains/severe/record.go'); s = p.read_text()
old = 'func Stamp(t time.Time, in *time.Location) string {\n\tif t.IsZero() {\n'
assert s.count(old) == 1, 'm760'
p.write_text(s.replace(old, 'func Stamp(t time.Time, in *time.Location) string {\n\tif t.IsZero() && false {\n', 1))
