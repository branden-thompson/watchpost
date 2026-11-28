import pathlib
# 0.19.6 B4.3 (design MS-4): the Settings view tells the map none of its rows
# is shown.
# Killed by TestTheLayersRowTogglesAndSaves and TestSetupGoldenEveryTab.
p = pathlib.Path('modes/tty/observer_map.go'); s = p.read_text()
old = '\t\tMapsOnShown: d.rowVisible(rowMapsOn), LayersShown: d.rowVisible(rowMapLayers), DetailLevelShown: d.rowVisible(rowMapDetailLevel),\n'
assert s.count(old) == 1, 'mMS5'
p.write_text(s.replace(old, '\t\tMapsOnShown: false, LayersShown: false, DetailLevelShown: false,\n', 1))
