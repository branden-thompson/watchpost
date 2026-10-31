import pathlib
# 0.19.0 D-210: the station every row shares is said once, in the status block.
# Killed by TestIdenticalTargetsShareALine.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tif !once {\n\t\t\tsay += "; " + d.propStationWords(a.Nearest)\n'
assert old in s, 'm408'
p.write_text(s.replace(old, '\t\tif !once || true {\n\t\t\tsay += "; " + d.propStationWords(a.Nearest)\n'))
