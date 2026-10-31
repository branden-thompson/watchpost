import pathlib
# 0.19.0 D-194: a window given a row has its top there.
# Killed by TestOverlayAtPlacesTheWindowsTopRow.
p = pathlib.Path('platform/render/panel.go'); s = p.read_text()
old = '\tif y < 0 {\n'
assert old in s, 'm249'
p.write_text(s.replace(old, '\tif y < 1<<20 {\n'))
