# 0.14.0 DR-3 / 0.16.0 P3. Remove airOnce's "the LANE is busy" guard: a second
# card takes the air over one already reading on the same lane.
#
# RE-ANCHORED AT D-82, where the guard moved BELOW `Next` and became a question
# about one lane. Two cards on the air is no longer a defect by itself — the rail
# speaks over the programme and the programme holds underneath (D-24) — but two
# on ONE lane is still two voices, and that is what this removes the guard for.
# The property test counts the OnAir states PER LANE for the same reason it
# counted them at all: Lineup.OnAir's own invariant makes a violation return
# "nobody", which would read as an idle station.
import pathlib
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tif _, busy := d.lineup.OnAir(track); busy {\n\t\treturn d, nil, false\n\t}\n"
new = "\tif _, busy := d.lineup.OnAir(track); false && busy {\n\t\treturn d, nil, false\n\t}\n"
assert old in s, "m105"
p.write_text(s.replace(old, new, 1))
