import pathlib
# 0.19.5 Q2.4 (RN-5): a tide is read in the location's zone.
# Killed by TestAMarineReportReadsTidesInTheLocationsZone.
p = pathlib.Path('app/marine.go'); s = p.read_text()
old = '\t\t\tr.TZ = z\n'
assert s.count(old) == 1, 'm688'
p.write_text(s.replace(old, '\t\t\t_ = z\n', 1))
