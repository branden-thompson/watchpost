import pathlib
# 0.19.0 W10.3 (D-208, D-218): a spot at exactly +13 dB normalised is open (D-91: at least).
# Killed by TestTheStrongestSpotInTheWindowDecides.
p = pathlib.Path('tools/reachscore/scenarios.go'); s = p.read_text()
old = 'if best >= voiceDB {'
assert old in s, 'm456'
p.write_text(s.replace(old, 'if best > voiceDB {', 1))
