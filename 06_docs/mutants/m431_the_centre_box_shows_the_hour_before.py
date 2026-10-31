import pathlib
# 0.19.0 P-6: the Centre box is the answer for what is shown now.
# Killed by TestTheCentreBoxIsForWhatIsShown.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\tif key, ok := d.propCentreWanted(); !ok || key != c.key { // the answer for what is shown now'
assert old in s, 'm431'
p.write_text(s.replace(old, '\tif key, ok := d.propCentreWanted(); !ok || key.made.computed != c.key.made.computed { // the answer for what is shown now'))
