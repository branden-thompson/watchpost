import pathlib
# 0.19.5 Q4.2 (PF-7, D-22): the masthead memo keys on the providers' counts the API summary gives.
# Killed by TestTheMastheadKeyCoversEverythingTheMastheadShows.
p = pathlib.Path('modes/tty/masthead.go'); s = p.read_text()
old = 'version: version, counts: countProviders(snap),'
assert s.count(old) == 1, 'm745'
p.write_text(s.replace(old, 'version: version,', 1))
