import pathlib
# 0.19.0 D-213, D-39 (R6T-1): the app's end does not stop an update in flight, so a quit during GIRO's burst keeps no state.
# Killed by TestTheAppsEndStopsAnUpdateAndKeepsItsState.
p = pathlib.Path('app/mapprop.go'); s = p.read_text()
old = '\t\t\tdefer context.AfterFunc(lp.ctx, stop)() //'
assert s.count(old) == 1, 'm575'
p.write_text(s.replace(old, '\t\t\t_ = context.AfterFunc //', 1))
