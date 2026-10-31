import pathlib
# 0.19.0 D-27, RC-5: a find in flight as the mode's session ends is left running, its answer landing past the mode.
# Killed by TestLeavingTheModeDropsAFindInFlight.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\td = d.dropFind() // a found place is not kept past the mode, nor one found after it (D-27)\n'
assert s.count(old) == 1, 'm541'
p.write_text(s.replace(old, '', 1))
