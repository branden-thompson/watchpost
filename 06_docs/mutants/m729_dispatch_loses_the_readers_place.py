import pathlib
# 0.19.5 Q3.1 (PF-1): dispatch, returning the Dashboard itself, still keeps a
# reader's place in the Propagation mode's words (D-211).
# Killed by TestARefreshKeepsTheReadersPlace.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = '\treturn next.followReader(msg, place, keep), nil, cmd, true\n'
assert s.count(old) == 1, 'm729'
p.write_text(s.replace(old, '\t_, _ = place, keep\n\treturn next, nil, cmd, true\n', 1))
