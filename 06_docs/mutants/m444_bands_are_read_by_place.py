import pathlib
# 0.19.0 L-F8: an answer's bands are named by the band each carries.
# Killed by TestBandsAreReadByName.
p = pathlib.Path('platform/propanswers/words.go'); s = p.read_text()
old = '\t\t\t\tnums = append(nums, strings.TrimSuffix(b.Band.String(), "m"))\n'
assert old in s, 'm444'
p.write_text(s.replace(old, '\t\t\t\tnums = append(nums, strings.TrimSuffix(ionomaps.Band(len(nums)+1).String(), "m"))\n'))
