import pathlib
# Stop takes whatever is on the air off it, including an alert mid-read.
#
# RE-ANCHORED AT D-82. The exemption used to be a check — "unless this card is on
# the rail" — and the mutant narrowed it to nothing. It is the ARGUMENT now:
# `silenceTheProgramme` asks the main track for its own card, so the way to break
# the rule is to ask the wrong lane. That is a better anchor than the old one,
# because the rule and the line are the same thing again.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "\tcard, live := d.lineup.OnAir(MainTrack)"
new = "\tcard, live := d.lineup.OnAir(AlertRail)"
assert old in s, "mA2"
p.write_text(s.replace(old, new, 1))
