import pathlib
# 0.19.0 D-213, D-39 (R7T-3): the app's end waits 2 ms for an update in flight, too little for a burst cut short to build its field and keep GIRO's state.
# Killed by TestTheAppsEndStopsAnUpdateAndKeepsItsState.
p = pathlib.Path('app/mapprop.go'); s = p.read_text()
old = 'const settleBound = 2 * time.Second'
assert s.count(old) == 1, 'm577'
p.write_text(s.replace(old, 'const settleBound = 2 * time.Millisecond', 1))
