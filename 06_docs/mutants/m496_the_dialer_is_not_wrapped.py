import pathlib
# 0.19.0 G1 (DN-12): the client's connections are handed up uncounted.
# Killed by TestThePropagationClientCountsItsWireBytes.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\t\treturn countedConn{Conn: c, read: &p.wire}, nil\n'
assert s.count(old) == 1, 'm496'
p.write_text(s.replace(old, '\t\treturn c, nil\n', 1))
