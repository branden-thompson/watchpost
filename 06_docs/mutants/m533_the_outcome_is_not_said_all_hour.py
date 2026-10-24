import pathlib
# 0.19.0 D-247: how GIRO's last burst ended is said only while it is paused, not all hour.
# Killed by TestTheLibrarysRealStatesAreSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = 'if w := giroWords(s.Inputs.GIRO); w != "" && s.Reason != ionomaps.GIROPaused {'
assert s.count(old) == 1, 'm533'
p.write_text(s.replace(old, 'if w := giroWords(s.Inputs.GIRO); w != "" && s.Reason == ionomaps.Unfinished {', 1))
