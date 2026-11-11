import pathlib
# 0.19.5 Q2.1 (RN-1): a line the bars follow plays through Preview.
# Killed by TestAVizClipDrivesTheBarsAndATakeoverDoesNot.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\t\t_ = d.engine.Preview(c.rate, bytes.NewReader(c.pcm)) // the bars follow it\n'
assert s.count(old) == 1, 'm677'
p.write_text(s.replace(old, '\t\t_ = d.engine.PreviewAside(c.rate, bytes.NewReader(c.pcm)) // the bars follow it\n', 1))
