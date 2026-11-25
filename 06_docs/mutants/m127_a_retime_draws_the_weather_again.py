import pathlib
# W2.2 (FR-1.4): a re-timing draws the mode's layers, none of the weather's in the Propagation mode.
# Killed by TestEachModeDrawsOnlyItsLayers.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = '\treturn mm.setFeed(mm.feedForLayers(*mm.mapPane.feed, e), e)'
assert old in s, "m127"
p.write_text(s.replace(old, '\treturn mm.setFeed(*mm.mapPane.feed, e)'))
