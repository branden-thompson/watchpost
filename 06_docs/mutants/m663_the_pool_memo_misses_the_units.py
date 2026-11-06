import pathlib
# 0.19.5 D-21: a units change with the console up redraws the memoised pool.
# Killed by TestAUnitsChangeRedrawsTheConsolesPool.
p = pathlib.Path('modes/tty/broadcaster_memo.go'); s = p.read_text()
old = 'theme: render.ThemeGeneration(), units: b.units, clock: b.clockFmt,'
assert s.count(old) == 1, 'm663'
p.write_text(s.replace(old, 'theme: render.ThemeGeneration(), clock: b.clockFmt,', 1))
