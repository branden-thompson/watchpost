import pathlib
# 0.19.0 R-2.6, FR-1.15: each row names the limit that decided it.
# Killed by TestBestBandsNameTheirLimit.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\t\tupper = "foF2"\n'
assert old in s, 'm197'
p.write_text(s.replace(old, '\t\t\tupper = "MUF"\n'))
