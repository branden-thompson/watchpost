import pathlib
# 0.19.0 D-193: the selected place standing in for a transmitter says so.
# Killed by TestTheOriginFollowsTheUIMode.
# `_ = frame.NoTransmitter` keeps the file's one use of internal/frame, so the
# import stands and the first line changed is the function's (0.19.6 A-56).
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '\t\t\treturn propPlace{at: ionomaps.LatLon{Lat: station.Lat, Lon: station.Lon}, name: station.Label + " " + frame.NoTransmitter, ok: true}'
assert s.count(old) == 1, 'm241'
p.write_text(s.replace(old, '\t\t\t_ = frame.NoTransmitter\n\t\t\treturn propPlace{at: ionomaps.LatLon{Lat: station.Lat, Lon: station.Lon}, name: station.Label, ok: true}'))
