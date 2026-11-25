import pathlib
# 0.19.6 B4.3 (design MS-1's S8 to S10, A-31): the save the map asks for joins
# the batch last, wherever the map's code put it.
# Killed by TestTheModeSwitchIsSavedFirst and TestAMenuSwitchIsSavedFirst.
p = pathlib.Path('modes/tty/observer_map.go'); s = p.read_text()
old = '\t\tif s.SaveLast {\n\t\t\tcmd = tea.Batch(cmd, save)\n'
assert s.count(old) == 1, 'mMS3'
p.write_text(s.replace(old, '\t\tif true {\n\t\t\tcmd = tea.Batch(cmd, save)\n', 1))
