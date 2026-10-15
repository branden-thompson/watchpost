import pathlib
# 0.19.0 D-73, D-193: a disturbed band is degraded, still usable.
# Killed by TestTheTargetSaysWhatIsUsable.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '(b.Status == ionomaps.Open || b.Status == ionomaps.Disturbed)'
assert old in s, 'm245'
p.write_text(s.replace(old, 'b.Status == ionomaps.Open'))
