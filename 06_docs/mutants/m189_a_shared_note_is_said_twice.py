import pathlib
# D-172: a note several hosts carry is said once.
# Killed by TestTheGroupsStandApartAndANoteIsSaidOnce.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\t\tif said[w] {\n'
assert old in s, 'm189'
p.write_text(s.replace(old, '\t\tif said[w] && false {\n'))
