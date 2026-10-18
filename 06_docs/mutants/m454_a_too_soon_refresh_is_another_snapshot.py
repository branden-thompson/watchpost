import pathlib
# 0.19.0 P-4: a refresh answered too soon keeps the answers and hours ahead of the snapshot held.
# Killed by TestARefreshAnsweredTooSoonKeepsItsAnswers.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\treturn held.Computed.Equal(got.Computed) && held.Noise == got.Noise && held.Inputs.NoStations == got.Inputs.NoStations\n'
assert old in s, 'm454'
p.write_text(s.replace(old, '\treturn false && held.Computed.Equal(got.Computed)\n'))
