import pathlib
# 0.19.0 G1: the counters record leaves out the propagation client's wire bytes.
# Killed by TestThePropagationClientsWireBytesReachTheCounters.
p = pathlib.Path('app/dump.go'); s = p.read_text()
old = '\trec.PropagationWireBytes = src.propagation.wireBytes()\n'
assert s.count(old) == 1, 'm497'
p.write_text(s.replace(old, '', 1))
