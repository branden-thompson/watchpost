import pathlib
# The store reads a roll-up's name one way: a month's parts are numbered from 1.
# Killed by TestARollUpsNameIsReadOneWay.
p = pathlib.Path('platform/history/history.go'); s = p.read_text()
old = '\tif err != nil || n < 1 {'
assert old in s, 'm178'
p.write_text(s.replace(old, '\tif err != nil {'))
