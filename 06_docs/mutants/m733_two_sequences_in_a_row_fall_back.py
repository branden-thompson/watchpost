import pathlib
# 0.19.5 Q3.2 (PF-2): an ESC beside a sequence is the next sequence's first byte
# and stands alone, so the frames' rows, full of sequences side by side, are
# summed and not stripped.
# Killed by TestWidthAllocBudget.
p = pathlib.Path('platform/render/text.go'); s = p.read_text()
old = '\tcase r == plaintext.Esc:\n'
assert s.count(old) == 1, 'm733'
p.write_text(s.replace(old, '\tcase r == plaintext.Esc && false:\n', 1))
