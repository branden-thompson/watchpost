import pathlib
# 0.19.5 Q2.1 (RN-1): a spent clip is said only when a read was on the air.
# Killed by TestASpentClipIsSaidOnlyWhenAReadWasOnTheAir.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\tif d.abandonRead != nil && !d.abandonRead("clip-spent") {\n'
assert s.count(old) == 1, 'm679'
p.write_text(s.replace(old, '\tif d.abandonRead != nil && false && !d.abandonRead("clip-spent") {\n', 1))
