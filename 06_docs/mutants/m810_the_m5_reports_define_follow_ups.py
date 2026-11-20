import pathlib
# 0.19.5 D-38: the mention passes read the blind traces and their keys
# (readable), so an ID they list as unresolvable resolves to that very list.
# Killed by TestAFollowUpNoTableDefinesIsIndexedByItsMention.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\t\tif !aboutIDs(p) {\n'
assert s.count(old) == 1, 'm810'
p.write_text(s.replace(old, '\t\tif true {\n', 1))
