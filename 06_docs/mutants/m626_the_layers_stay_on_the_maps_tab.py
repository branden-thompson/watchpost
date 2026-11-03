import pathlib
# 0.19.3 D-364: MAP - DETAIL and MAP - LAYERS are the Map Layers tab's.
# Killed by TestTheMapLayersTabHoldsDetailAndLayers.
p = pathlib.Path('modes/tty/setup_tabs.go'); s = p.read_text()
old = "\tcase groupLayers, groupMapLayers:\n\t\treturn tabMapLayers\n"
assert s.count(old) == 1, 'm626'
p.write_text(s.replace(old, "\tcase groupLayers, groupMapLayers:\n\t\treturn tabMaps\n", 1))
