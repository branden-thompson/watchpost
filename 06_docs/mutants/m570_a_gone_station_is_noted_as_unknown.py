import pathlib
# 0.19.0 D-272 (R4D-3): a 410, which D-247 and D-272 name, is noted as a status it is not known to send.
# Killed by TestAnUnnamedStatusIsNotedOnce.
p = pathlib.Path('domains/propagation/propagation.go'); s = p.read_text()
old = '404: true, 410: true, 429: true}'
assert s.count(old) == 1, 'm570'
p.write_text(s.replace(old, '404: true, 429: true}', 1))
