import pathlib
# 0.19.5 Q5.4 (D-17, DR-7a): a bearing below zero is taken round the rose.
# Killed by TestCompassIndexNormalisesANegativeBearing.
p = pathlib.Path('platform/geo/geo.go'); s = p.read_text()
old = '\tdeg = math.Mod(deg, 360)\n\tif deg < 0 {\n\t\tdeg += 360\n\t}\n'
assert s.count(old) == 1, 'm764'
p.write_text(s.replace(old, '\tdeg = math.Mod(deg, 360)\n', 1))
