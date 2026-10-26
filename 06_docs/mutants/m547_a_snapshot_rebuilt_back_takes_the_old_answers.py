import pathlib
# 0.19.0 L-F3, R2C-1: every snapshot as kept has build 0, so one rebuilt and then rebuilt back at the same moment takes the answers worked out before.
# Killed by TestARebuildAndBackIsAnsweredAgain.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'return propMade{computed: s.Computed, built: d.mapPane.propBuilt, opts: '
assert s.count(old) == 1, 'm547'
p.write_text(s.replace(old, 'return propMade{computed: s.Computed, built: 0, opts: ', 1))
