import pathlib
# 0.19.0 AX-9: the refresh key lets a failed answer be asked again.
# Killed by TestAFailedAnswerSaysHowToAskAgain.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tif d.mapPane.propChart.err != nil { // an answer that failed'
assert old in s, 'm417'
p.write_text(s.replace(old, '\tif d.mapPane.propChart.err != nil && false { // an answer that failed'))
