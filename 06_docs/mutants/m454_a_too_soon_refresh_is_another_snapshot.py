import pathlib
# 0.19.0 P-4: a refresh answered too soon keeps the answers and hours ahead of the snapshot held.
# Killed by TestARefreshAnsweredTooSoonKeepsItsAnswers.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\treturn held.Computed.Equal(got.Computed) && madeWith(held, heldCorrect) == madeWith(got, gotCorrect)\n'
assert s.count(old) == 1, 'm454'
p.write_text(s.replace(old, '\treturn false && held.Computed.Equal(got.Computed)\n', 1))
