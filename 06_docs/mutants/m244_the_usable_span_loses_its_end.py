import pathlib
# 0.19.0 D-193: the usable bands are said as a span.
# Killed by TestTheTargetSaysWhatIsUsable.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '"usable " + u[0] + "-" + u[len(u)-1]'
assert old in s, 'm244'
p.write_text(s.replace(old, '"usable " + u[0]'))
