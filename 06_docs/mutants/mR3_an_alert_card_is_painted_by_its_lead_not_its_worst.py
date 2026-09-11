import pathlib
# A burst is painted by the FIRST alert it lists rather than the most severe one,
# so an Emergency inside a burst led by a Watch is drawn as a Watch. A card reads
# several alerts (MVS-D-77) and its colour is a promise about all of them.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\tif !found || spec.ReadRank < category.Of(worst).ReadRank {"
new = "\t\tif !found {"
assert old in s, "mR3"
p.write_text(s.replace(old, new, 1))
