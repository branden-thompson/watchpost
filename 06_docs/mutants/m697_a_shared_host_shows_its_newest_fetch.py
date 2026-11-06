import pathlib
# 0.19.5 Q2.6 (RN-7): a shared host shows its oldest fetch.
# Killed by TestASharedHostShowsItsWorstFeedAndItsOldestFetch.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '(!p.FetchedAt.IsZero() && p.FetchedAt.Before(r.fetchedAt))'
assert s.count(old) == 1, 'm697'
p.write_text(s.replace(old, '(!p.FetchedAt.IsZero() && p.FetchedAt.After(r.fetchedAt))', 1))
