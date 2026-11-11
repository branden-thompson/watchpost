import pathlib
# 0.19.5 Q3.2 (PF-2): braille never joins a grapheme cluster, so the map's rows
# of coloured dots are summed and not stripped.
# Killed by TestWidthAllocBudget.
p = pathlib.Path('platform/render/text.go'); s = p.read_text()
old = '\tcase r >= 0x2800 && r <= 0x28ff:\n'
assert s.count(old) == 1, 'm812'
p.write_text(s.replace(old, '\tcase r >= 0x2800 && r <= 0x28ff && false:\n', 1))
