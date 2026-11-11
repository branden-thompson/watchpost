import pathlib
# 0.19.5 Q2.6 (RN-7): a shared host shows its unhealthiest feed.
# Killed by TestASharedHostShowsItsWorstFeedAndItsOldestFetch.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\t\t\tif r.status == "" || worseStatus(p.Status, r.status) {\n'
assert s.count(old) == 1, 'm696'
p.write_text(s.replace(old, '\t\t\tif r.status == "" {\n', 1))
