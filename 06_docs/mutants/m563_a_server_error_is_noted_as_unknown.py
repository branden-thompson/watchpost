import pathlib
# 0.19.0 D-272 (R3L-5, R3C-3): a server error, which D-247 and D-260 name, is noted as a status the host is not known to send.
# Killed by TestAnUnnamedStatusIsNotedOnce.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '\tif status >= 500 && status <= 599 {\n'
assert s.count(old) == 1, 'm563'
p.write_text(s.replace(old, '\tif status >= 500 && status <= 499 {\n', 1))
