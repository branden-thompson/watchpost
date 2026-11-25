import pathlib
# 0.19.6 B4.3 (design MS-1's split sites): the Dashboard does what the map
# asked and never calls the map's own step after it.
# Killed by TestClosingTheAcknowledgementRecordsItSeen and
# TestTheMapFollowsTheSelection.
p = pathlib.Path('modes/tty/observer_map.go'); s = p.read_text()
old = '\tif w.Then == mapStepNone {\n\t\treturn d, cmd\n\t}\n'
assert s.count(old) == 1, 'mMS4'
p.write_text(s.replace(old, '\tif true {\n\t\treturn d, cmd\n\t}\n', 1))
