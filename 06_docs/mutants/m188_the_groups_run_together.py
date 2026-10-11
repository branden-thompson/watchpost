import pathlib
# D-171: a blank row stands between the Status window's groups.
# Killed by TestTheGroupsStandApartAndANoteIsSaidOnce.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\t\t\tout = append(out, "") // a blank row between the groups (D-171)\n'
assert old in s, 'm188'
p.write_text(s.replace(old, '\t\t\tout = append(out, out[len(out)-1]) // a blank row between the groups (D-171)\n'))
