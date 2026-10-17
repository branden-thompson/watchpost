import pathlib
# 0.19.0 D-187: the service asks the library with the host's options.
# Killed by TestTheHostsOptionsReachTheLibrary.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\t\tsnap, err := s.lib.Update(uctx, opts)\n'
assert old in s, 'm233'
p.write_text(s.replace(old, '\t\tsnap, err := s.lib.Update(uctx, ionomaps.UpdateOptions{})\n'))
