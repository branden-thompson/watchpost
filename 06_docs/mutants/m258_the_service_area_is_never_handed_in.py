import pathlib
# 0.19.0 D-195: the service area is handed to the library.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = 'mm.reconcile(mm.mapPane.zoneGiven, mm.zoneOverlays(e),'
assert old in s, 'm258'
p.write_text(s.replace(old, 'mm.reconcile(mm.mapPane.zoneGiven, nil,'))
