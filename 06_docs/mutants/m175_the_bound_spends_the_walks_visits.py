import pathlib
# D-143: a dataset that cannot get under its byte bound leaves the other datasets' expiry to run.
# Killed by TestABoundThatCannotBeMetStarvesNoOtherDataset.
p = pathlib.Path('platform/history/prune.go'); s = p.read_text()
old = "\tbounds := 0\n\tfor _, d := range sets { // a handful (P10-02)\n\t\tremovals += s.holdToBound(d, now, &bounds, maxPruneRemovals-removals)"
assert old in s, 'm175'
p.write_text(s.replace(old, "\tfor _, d := range sets { // a handful (P10-02)\n\t\tremovals += s.holdToBound(d, now, &visits, maxPruneRemovals-removals)"))
