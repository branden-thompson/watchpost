import pathlib
# 0.19.5 D-8: the old default, first in the list, always goes.
# Killed by TestSetupFoldsCoordinatesEqualToFourDecimals.
p = pathlib.Path('app/setup.go'); s = p.read_text()
old = '\t\t\tif i == 0 || snapshot.PlaceID('
assert s.count(old) == 1, 'm670'
p.write_text(s.replace(old, '\t\t\tif i < 0 || snapshot.PlaceID(', 1))
