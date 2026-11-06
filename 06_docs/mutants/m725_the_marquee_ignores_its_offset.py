import pathlib
# 0.19.5 Q3.5 (PF-5): the marquee walked in place starts its window at the offset.
# Killed by TestTheMarqueeWindowIsByteIdentical.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = '\tfor range ((offset % n) + n) % n { // bounded by the loop\'s length (P10-02)\n'
assert s.count(old) == 1, 'm725'
p.write_text(s.replace(old, '\tfor range ((offset % n) + n) % n / 2 { // bounded by the loop\'s length (P10-02)\n', 1))
