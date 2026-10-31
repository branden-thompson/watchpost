import pathlib
# 0.19.0 FR-2.3: the Broadcaster's local line names the service area.
# Killed by TestTheNearVerticalAnswersUseTheServiceRadius.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '\t\treturn "service area"\n'
assert old in s, 'm270'
p.write_text(s.replace(old, '\t\treturn "NVIS"\n'))
