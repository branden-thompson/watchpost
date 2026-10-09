import pathlib
# W2.3 (FR-1.3): the outside message names the Propagation key as bound.
# Killed by TestTheOutsideMessageNamesThePropagationKey.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\t\ttext += \" Press \" + keys[0] + \" for the Propagation mode, which covers the whole world.\""
assert old in s, "m125"
p.write_text(s.replace(old, "\t\ttext += \"\""))
