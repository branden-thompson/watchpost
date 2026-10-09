import pathlib
# W2.2 (FR-1.4, D-21): no weather layer is drawn in the Propagation mode.
# Killed by TestEachModeDrawsOnlyItsLayers.
p = pathlib.Path("modes/tty/map_prefs.go"); s = p.read_text()
old = "\tif d.mapMode() == modePropagation {\n\t\treturn false\n\t}\n\tif key == TemperatureLayer"
assert old in s, "m121"
p.write_text(s.replace(old, "\tif key == TemperatureLayer"))
