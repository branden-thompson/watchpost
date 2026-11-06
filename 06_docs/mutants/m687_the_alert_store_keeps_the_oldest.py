import pathlib
# 0.19.5 Q2.3 (RN-3): the alert store's cap evicts the oldest records.
# Killed by TestTheAlertStoreKeepsTheNewestRecordsAtItsCap.
p = pathlib.Path('app/alert_store.go'); s = p.read_text()
old = '\tsort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })\n\tfor _, e := range all[maxAlertRecords:]'
assert s.count(old) == 1, 'm687'
p.write_text(s.replace(old, '\tsort.Slice(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })\n\tfor _, e := range all[maxAlertRecords:]', 1))
