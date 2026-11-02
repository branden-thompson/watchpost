import pathlib
# 0.19.3 F-196 (A-107 (4)): under the Monochrome theme, Automatic hints the map no colour.
# Killed by TestTheMonochromeThemeDrawsTheMapWithoutColour.
p = pathlib.Path('modes/tty/map_depth.go'); s = p.read_text()
old = '\tif render.ThemeName() == render.MonochromeThemeName {\n'
assert s.count(old) == 1, 'm607'
p.write_text(s.replace(old, '\tif render.ThemeName() == render.MonochromeThemeName && false {\n', 1))
