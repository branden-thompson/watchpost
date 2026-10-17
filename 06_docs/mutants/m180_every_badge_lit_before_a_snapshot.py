import pathlib
# D-168: before any snapshot the sources' badges are dimmed, nothing having fed them.
# Killed by TestTheSourcesBadgesSayWhatFedTheSnapshot.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tshown := snap != nil && propHasField(*snap)\n'
assert old in s, 'm180'
p.write_text(s.replace(old, '\tshown := true || snap != nil && propHasField(*snap)\n'))
