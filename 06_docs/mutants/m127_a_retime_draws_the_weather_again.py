import pathlib
# W2.2 (FR-1.4): a re-timing draws the mode's layers, none of the weather's in the Propagation mode.
# Killed by TestEachModeDrawsOnlyItsLayers.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = "\treturn d.setFeed(d.feedForLayers(*d.mapPane.feed))"
assert old in s, "m127"
p.write_text(s.replace(old, "\treturn d.setFeed(*d.mapPane.feed)"))
