import pathlib
# 0.19.5 Q2.11 (RN minor): the sweep judges a temp file's age by the store's clock.
# Killed by TestTheSweepJudgesAgeByTheStoresClock.
p = pathlib.Path('platform/history/prune.go'); s = p.read_text()
old = "\tif now.Sub(info.ModTime()) > claimStale {"
assert s.count(old) == 1, 'm718'
p.write_text(s.replace(old, "\tif time.Since(info.ModTime()) > claimStale {", 1))
