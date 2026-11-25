import pathlib
# 0.19.6 B4.3 (design MS-2, A-31): the table both translations read holds the
# frequency entry where the finder is.
# Killed by TestTheMapsWindowsTranslateBothWays and
# TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/observer_map.go'); s = p.read_text()
old = '\tmapModalPropFind:  modalPropFind,\n'
assert s.count(old) == 1, 'mMS1'
p.write_text(s.replace(old, '\tmapModalPropFind:  modalPropFreq,\n', 1))
