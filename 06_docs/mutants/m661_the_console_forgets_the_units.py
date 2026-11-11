import pathlib
# 0.19.5 D-5 (DEF-2): the console draws in the units Setting.
# Killed by TestTheConsolesPoolWearsTheListenersUnits.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = 'render.Opts{Width: b.frameWidth(), Units: b.units, Clock: b.clockFmt,'
assert s.count(old) == 1, 'm661'
p.write_text(s.replace(old, 'render.Opts{Width: b.frameWidth(), Clock: b.clockFmt,', 1))
