import pathlib
# 0.19.0 G1 (D-141, DN-12): a connection's reads are not counted.
# Killed by TestThePropagationClientCountsItsWireBytes.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\tc.read.Add(int64(n))\n'
assert s.count(old) == 1, 'm495'
p.write_text(s.replace(old, '', 1))
