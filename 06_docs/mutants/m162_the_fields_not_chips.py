import pathlib
# D-165: foF2 and MUF(3000) are chips on the place's line.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tout = append(out, fmt.Sprintf("%sNOW · %s %.1f MHz · %s %.1f MHz", place, field("foF2"), fo, field("MUF(3000)"), muf))'
assert old in s, 'm162'
p.write_text(s.replace(old, '\tout = append(out, fmt.Sprintf("%sNOW · %s %.1f MHz · %s %.1f MHz", place, "foF2", fo, "MUF(3000)", muf))'))
