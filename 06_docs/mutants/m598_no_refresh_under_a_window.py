import pathlib
# 0.19.0 D-339: the refresh asks only with the map in front, so a chart left open never refreshes.
# Killed by TestTheModeRefreshesUnderAWindow.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '!d.mapPane.propSess.busy && d.mapShown() && d.modal != modalPropAck && d.mapMode() == modePropagation'
assert s.count(old) == 1, 'm598'
p.write_text(s.replace(old, '!d.mapPane.propSess.busy && d.modal == modalMap && d.mapMode() == modePropagation', 1))
