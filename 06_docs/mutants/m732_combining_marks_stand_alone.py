import pathlib
# 0.19.5 Q3.2 (PF-2): a combining mark beside a sequence joins the rune before it
# once the sequence is stripped, so it never counts as standing alone.
# Killed by TestWidthMatchesStripThenMeasure.
p = pathlib.Path('platform/render/text.go'); s = p.read_text()
old = 'r >= 0xa0 && r < 0x300 && r != 0xad:'
assert s.count(old) == 1, 'm732'
p.write_text(s.replace(old, 'r >= 0xa0 && r < 0x370 && r != 0xad:', 1))
