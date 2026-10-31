import pathlib
# 0.19.0 BUILD red team round 2 (D-205, FR-1.16): a reach stopped by the absorption or the circuit is said by that limit, never the MUF.
# Killed by TestTheReachSaysWhichLimitStopsIt.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\t\treturn "it reaches nowhere: " + words[0]'
assert old in s, 'm472'
p.write_text(s.replace(old, '\t\treturn "it reaches nowhere: above the MUF everywhere"', 1))
