import pathlib
# The ruling most likely to be "corrected" by someone reading the tab bar,
# where Disasters sits BELOW Warnings.
p = pathlib.Path("platform/category/category.go"); s = p.read_text()
s2 = s.replace('BandLabel: "Warnings", Rotation: 3, ReadRank: 3}', 'BandLabel: "Warnings", Rotation: 3, ReadRank: 2}', 1)
s2 = s2.replace('BandLabel: "Disasters", Rotation: 1, ReadRank: 2}', 'BandLabel: "Disasters", Rotation: 1, ReadRank: 3}', 1)
assert s2 != s, "m57"
p.write_text(s2)
