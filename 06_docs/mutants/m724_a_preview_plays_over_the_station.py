import pathlib
# 0.19.5 Q2.13 (RN minor): a preview is refused while the station holds the air.
# Killed by TestAPreviewIsRefusedWhileTheStationHoldsTheAir.
p = pathlib.Path('app/voices.go'); s = p.read_text()
old = '\t\td.voiceNote("preview is unavailable while the station holds the air")\n\t\treturn\n'
assert s.count(old) == 1, 'm724'
p.write_text(s.replace(old, '\t\td.voiceNote("preview is unavailable")\n\t\treturn\n', 1))
