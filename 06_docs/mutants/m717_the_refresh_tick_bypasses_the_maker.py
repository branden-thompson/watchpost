import pathlib
# 0.19.5 Q2.11 (RN-9): the Propagation refresh tick is made by the model's tick
# maker, so a test tells an armed tick from an answer by type, not by a clock.
# Killed by TestStationsNextAsksAgainAtOnce.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\treturn d, d.after(every, tick)\n'
assert s.count(old) == 1, 'm717'
p.write_text(s.replace(old, '\treturn d, tea.Tick(every, func(time.Time) tea.Msg { return tick })\n', 1))
