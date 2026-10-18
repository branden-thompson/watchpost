import pathlib
# 0.19.0 BUILD red team round 2 (L-F8, CN2-14): a row's glyph sits under the band it names.
# Killed by TestEachBandIsFoundByItsName.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\t\tif a.Band.String() == hb.name {'
assert old in s, 'm484'
p.write_text(s.replace(old, '\t\t\tif a.Band.String() != hb.name {', 1))
