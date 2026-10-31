import pathlib
# 0.19.0 D-73, D-193: a disturbed band is degraded, still usable.
# Killed by TestTheTargetSaysWhatIsUsable.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = '\t\tif b.Status == ionomaps.Open || b.Status == ionomaps.Disturbed {\n\t\t\tout = append(out, b.Band.String())'
assert old in s, 'm245'
p.write_text(s.replace(old, '\t\tif b.Status == ionomaps.Open {\n\t\t\tout = append(out, b.Band.String())'))
