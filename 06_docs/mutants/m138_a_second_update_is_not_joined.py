import pathlib
# 0.19.0 FR-4.8 (D-131): an update asked while one runs joins it.
# Killed by TestAnUpdateIsJoinedNotDoubled.
p = pathlib.Path("domains/propagation/propagation.go"); s = p.read_text()
old = "\tr := s.running\n\tif r == nil {"
assert old in s, "m138"
p.write_text(s.replace(old, "\tr := s.running\n\tif true {"))
