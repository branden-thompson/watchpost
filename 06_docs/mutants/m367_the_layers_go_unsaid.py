import pathlib
# 0.19.0 FR-3.3: the layers' sentences come before the alerts' early returns.
# Killed by TestALayerAddsItsSentence.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\tout := append([]string{mm.viewHeading(*loc, here, e)}, mm.layerWords(e)...)'
assert old in s, 'm367'
p.write_text(s.replace(old, '\tout := []string{mm.viewHeading(*loc, here, e)}'))
