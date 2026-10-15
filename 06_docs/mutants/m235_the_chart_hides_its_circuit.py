import pathlib
# 0.19.0 D-187: the chart says its circuit and noise.
# Killed by TestBestBandsNameTheirLimit.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = ' + " · 100 W, 0 dBi, +13 dB · " + strings.ToLower(propNoiseLabel(c.key.noise)) + " noise", ""}'
assert old in s, 'm235'
p.write_text(s.replace(old, ' + " · upper limits only", ""}'))
