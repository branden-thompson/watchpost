import pathlib
# D-168: SWPC's badge is dimmed when foF2 is the climatology's.
# Killed by TestTheSourcesBadgesSayWhatFedTheSnapshot.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '{"SWPC", shown && snap.Background.FoF2 == ionomaps.GloTEC}'
assert old in s, 'm181'
p.write_text(s.replace(old, '{"SWPC", shown}'))
