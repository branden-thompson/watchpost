import pathlib
# 0.19.0 FR-2.3: the Broadcaster's near-vertical answers are to the service radius.
# Killed by TestTheNearVerticalAnswersUseTheServiceRadius.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '\tif km := d.serviceKm(); d.mapRole().zone && km > 0 {\n'
assert old in s, 'm269'
p.write_text(s.replace(old, '\tif km := d.serviceKm(); false && km > 0 {\n'))
