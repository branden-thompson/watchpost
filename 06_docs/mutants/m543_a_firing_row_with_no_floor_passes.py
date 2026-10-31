import pathlib
# 0.19.0 D-277: a firing density row whose reason states no floor passes, unnamed.
# Killed by TestARowUnderItsFloorFails.
p = pathlib.Path('tools/p10floor/main.go'); s = p.read_text()
old = '\t\t\tfloorless = append(floorless, fmt.Sprintf('
assert s.count(old) == 1, 'm543'
p.write_text(s.replace(old, '\t\t\t_ = append(floorless, fmt.Sprintf(', 1))
