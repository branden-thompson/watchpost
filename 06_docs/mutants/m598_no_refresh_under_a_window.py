import pathlib
# 0.19.0 D-339: the refresh asks only with the map in front, so a chart left open never refreshes.
# Killed by TestTheModeRefreshesUnderAWindow.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '!mm.mapPane.propSess.busy && e.Shown && e.Window != mapModalPropAck && mm.mapMode(e) == modePropagation'
assert s.count(old) == 1, 'm598'
p.write_text(s.replace(old, '!mm.mapPane.propSess.busy && e.Window == mapModalMap && mm.mapMode(e) == modePropagation', 1))
