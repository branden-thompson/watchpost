import pathlib
# 0.19.5 Q3.1 (PF-1): the Router keeps the Dashboard its update returns when
# Observer is on screen; the random-message trace sees a frame that never moves.
# Killed by TestRouterUpdateIsTheSameUnderRandomMessages.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\tcase SurfaceObserver:\n\t\tvar cmd tea.Cmd\n\t\tr.observer, cmd = r.observer.update(msg)\n'
assert s.count(old) == 1, 'm740'
p.write_text(s.replace(old, '\tcase SurfaceObserver:\n\t\tvar cmd tea.Cmd\n\t\t_, cmd = r.observer.update(msg)\n', 1))
