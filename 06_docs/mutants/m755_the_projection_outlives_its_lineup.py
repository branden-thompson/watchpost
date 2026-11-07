import pathlib
# 0.19.5 Q4.5 (D-22): the projection is worked out again for each line-up the console is handed.
# Killed by TestTheConsolesProjectionFollowsItsLineup.
p = pathlib.Path('modes/tty/broadcaster_memo.go'); s = p.read_text()
old = '\tif !s.ok || s.gen != b.lineupGen {\n'
assert s.count(old) == 1, 'm755'
p.write_text(s.replace(old, '\tif !s.ok {\n', 1))
