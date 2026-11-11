import pathlib
# 0.19.5 Q5.6 (DR-10): the spoken description measures the view as the title
# does, across its centre's latitude.
# Killed by TestTheTitleAndTheDescriptionNameOneArea.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = 'd.cfg.MapAreaName(centre, viewWidthKm(v, centre.Lat))'
assert s.count(old) == 1, 'm772'
p.write_text(s.replace(old, 'd.cfg.MapAreaName(centre, viewWidthKm(v, 0))', 1))
