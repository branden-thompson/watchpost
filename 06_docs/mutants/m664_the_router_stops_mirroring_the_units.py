import pathlib
# 0.19.5 D-21: the Router mirrors the units and clock into the console on every update.
# Killed by TestTheConsolesPoolWearsTheListenersUnits.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\t\tout.broadcaster.units, out.broadcaster.clockFmt = out.observer.units, out.observer.clockFmt\n'
assert s.count(old) == 1, 'm664'
p.write_text(s.replace(old, '', 1))
