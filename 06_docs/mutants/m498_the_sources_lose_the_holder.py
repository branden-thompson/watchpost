import pathlib
# 0.19.0 G1: the diagnostic sources are not handed the propagation holder.
# Killed by TestThePropagationClientsWireBytesReachTheCounters.
p = pathlib.Path('app/stats.go'); s = p.read_text()
old = ' propagation: &lp.propagation,\n'
assert s.count(old) == 1, 'm498'
p.write_text(s.replace(old, '\n', 1))
