import pathlib
# D-170: a shared host's counters are the station's and the map's together.
# Killed by TestTheStatusWindowGroupsHostsByWhatTheyServe.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = 'Attempts: a.Attempts + b.Attempts,'
assert old in s, 'm185'
p.write_text(s.replace(old, 'Attempts: a.Attempts,'))
