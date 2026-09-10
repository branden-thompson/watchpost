# 0.14.0 DR-3 / 0.16.0 P3. Remove airOnce's "the air is busy" guard: a second
# card takes the air over one already reading.
#
# TWO CARDS ON THE AIR IS TWO VOICES, and it is the defect the whole schedule
# exists to make impossible. The property test counts the OnAir states directly
# rather than asking Lineup.OnAir, because that method's own invariant makes a
# violation return "nobody" — which would read as an idle station.
import pathlib
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tif _, busy := d.lineup.OnAir(); busy {\n\t\treturn d, nil, false\n\t}\n\tnext, track, ok := d.lineup.Next()"
assert old in s, "m105"
p.write_text(s.replace(old, "\tif _, busy := d.lineup.OnAir(); false && busy {\n\t\treturn d, nil, false\n\t}\n\tnext, track, ok := d.lineup.Next()"))
