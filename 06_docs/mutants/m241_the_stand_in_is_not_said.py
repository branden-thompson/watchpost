import pathlib
# 0.19.0 D-193: the selected place standing in for a transmitter says so.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'name: station.Label + " " + frame.NoTransmitter, ok: true}'
assert old in s, 'm241'
s = s.replace(old, 'name: station.Label, ok: true}')
# The line was the file's one use of internal/frame, so the import goes with it.
imp = '\t"github.com/branden-thompson/watchpost/modes/tty/internal/frame"\n'
assert s.count(imp) == 1 and s.count('frame.') == 0, 'm241 import'
p.write_text(s.replace(imp, ''))
