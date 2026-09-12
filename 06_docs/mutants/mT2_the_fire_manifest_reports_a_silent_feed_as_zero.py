import pathlib
# The manifest stops asking whether a fire feed ANSWERED and reports its count
# either way, so a provider that was down reads as "0 Hotspots" on the one line
# the operator decides from. FireReport's own rule: "a zero count is only a fact
# when its own feed did."
#
# It looks exactly like a measurement, which is what makes it worse than a blank.
p = pathlib.Path("domains/radio/synth/manifest.go"); s = p.read_text()
old = "\tif f.HotspotsKnown {"
new = "\tif true {"
assert old in s, "mT2"
p.write_text(s.replace(old, new, 1))
