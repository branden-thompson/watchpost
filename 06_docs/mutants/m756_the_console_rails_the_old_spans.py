import pathlib
# 0.19.5 Q4.5 (D-22): the console's memo rails the spans it has just built,
# not the ones it held before.
# Killed by TestTheConsolesTableRowsAreTheRowsItWouldDraw.
p = pathlib.Path('modes/tty/broadcaster_memo.go'); s = p.read_text()
old = 'm.ok, m.key, m.sched, m.pool, m.rows = true, key, sched, pool, b.railTables(sched, pool)'
assert s.count(old) == 1, 'm756'
p.write_text(s.replace(old, 'm.rows = b.railTables(m.sched, m.pool)\n\tm.ok, m.key, m.sched, m.pool = true, key, sched, pool', 1))
