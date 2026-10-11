import pathlib
# D-170: a host two features use is COMMON DATA.
# Killed by TestTheStatusWindowGroupsHostsByWhatTheyServe.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\tcase n > 1:\n\t\treturn groupCommon\n'
assert old in s, 'm184'
p.write_text(s.replace(old, '\tcase n > 1:\n\t\treturn groupOther\n'))
