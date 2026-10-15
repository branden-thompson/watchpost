import pathlib
# 0.19.0 D-195: the service area is handed to the library.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = 'd.reconcile(d.mapPane.zoneGiven, d.zoneOverlays(),'
assert old in s, 'm258'
p.write_text(s.replace(old, 'd.reconcile(d.mapPane.zoneGiven, nil,'))
