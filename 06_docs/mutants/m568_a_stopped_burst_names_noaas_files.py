import pathlib
# 0.19.0 D-286: an update stopped while GIRO's burst was its work is said as NOAA's files not coming.
# Killed by TestAStoppedBurstNamesGIRO.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\t\tnd.mapPane.propSess.burst = true'
assert s.count(old) == 1, 'm568'
p.write_text(s.replace(old, '\t\t\tnd.mapPane.propSess.burst = false', 1))
