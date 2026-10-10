import pathlib
# D-160: foF2's class below 1.8 MHz keys no band.
# Killed by TestThePropagationLegendIsBandOverMHz.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tcase n + 1:\n\t\treturn classes[1:], true'
assert old in s, 'm153'
p.write_text(s.replace(old, '\tcase n + 1:\n\t\treturn classes[:n], true'))
