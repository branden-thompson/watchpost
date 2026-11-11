import pathlib
# 0.19.5 Q2.11 (RN minor): the sweep judges a file's age by the store's clock.
# Killed by TestTheSweepJudgesAgeByTheStoresClock.
p = pathlib.Path('platform/history/claim.go'); s = p.read_text()
old = 'return now.Sub(info.ModTime()) > claimStale'
assert s.count(old) == 1, 'm719'
p.write_text(s.replace(old, 'return time.Since(info.ModTime()) > claimStale', 1))
