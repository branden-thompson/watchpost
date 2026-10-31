import pathlib
# D-167: the badge reads RADIO FREQ. PROPAGATION.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_prop.go'); s = p.read_text()
old = '\tpropagationLabel  = "RADIO FREQ. PROPAGATION" // D-167'
assert old in s, 'm165'
p.write_text(s.replace(old, '\tpropagationLabel  = "PROPAGATION" // D-167'))
