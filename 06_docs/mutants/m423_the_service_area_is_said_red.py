import pathlib
# 0.19.0 AX-12: the service area's sentence names no colour.
# Killed by TestTheServiceAreaIsSaid.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '\treturn "The circle around " + loc.Label'
assert old in s, 'm423'
p.write_text(s.replace(old, '\treturn "The red circle around " + loc.Label'))
