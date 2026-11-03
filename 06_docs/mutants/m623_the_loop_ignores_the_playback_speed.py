import pathlib
# 0.19.3 D-361: the radar's loop plays at the speed chosen.
# Killed by TestThePlaybackSpeedReachesTheLoop.
p = pathlib.Path('modes/tty/map_radar.go'); s = p.read_text()
old = "\tstep := d.mapPlayback\n"
assert s.count(old) == 1, 'm623'
p.write_text(s.replace(old, "\tstep := time.Second\n", 1))
