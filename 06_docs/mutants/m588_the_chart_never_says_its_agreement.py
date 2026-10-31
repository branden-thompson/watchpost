import pathlib
# 0.19.0 D-330: the chart's measured agreement is never drawn.
# Killed by TestTheChartSaysItsMeasuredAgreement.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tpropShadeLegend(o.ASCII), // D-198\n\t\tpropAgreementLine)        // D-330\n'
assert s.count(old) == 1, 'm588'
p.write_text(s.replace(old, '\t\tpropShadeLegend(o.ASCII)) // D-198\n', 1))
