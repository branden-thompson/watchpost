import pathlib
# 0.19.0 D-286 (R4C-3): the burst's flag outlives the session it was set in, so NOAA's round after the mode is reopened is said as GIRO's burst.
# Killed by TestAStoppedBurstNamesGIRO.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\td.mapPane.propSess = propSession{}\n'
assert s.count(old) == 1, 'm572'
p.write_text(s.replace(old, '\td.mapPane.propSess = propSession{burst: d.mapPane.propSess.burst}\n', 1))
