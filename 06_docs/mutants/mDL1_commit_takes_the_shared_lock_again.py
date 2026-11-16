import pathlib
# 0.19.6 B0.11 (D-16, design AP-3.6): commit's section asks for the station's
# pool through currentPool, which takes lp.mu again - a deadlock on itself.
# Killed by TestCommitAndStopAllFinishUnderTheSharedLock and
# TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage.
p = pathlib.Path('app/dashboard.go'); s = p.read_text()
old = '\tlp.watchRefs = append([]snapshot.LocationRef(nil), watch...)'
assert s.count(old) == 1, 'mDL1'
p.write_text(s.replace(old, '\t_ = lp.currentPool()\n' + old, 1))
