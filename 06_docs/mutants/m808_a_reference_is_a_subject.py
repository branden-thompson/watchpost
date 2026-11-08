import pathlib
# 0.19.5 D-37: the subject is the first cell, references and all, so an ID's
# row reads "W1.13, W3.7" instead of what it says.
# Killed by TestAPriorityStillDefinesAndAReferenceIsNoSubject.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\t\tif word.MatchString(c) {\n'
assert s.count(old) == 1, 'm808'
p.write_text(s.replace(old, '\t\tif true {\n', 1))
