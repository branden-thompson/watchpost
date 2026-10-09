import pathlib
# W2.3 (FR-1.3): a place in no region opens the Propagation mode.
# Killed by TestAPlaceOutsideEveryRegionOpensThePropagationMode.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\tcase modePropagation:\n\t}\n\treturn false\n}"
assert old in s, "m126"
p.write_text(s.replace(old, "\tcase modePropagation:\n\t\treturn true\n\t}\n\treturn false\n}"))
