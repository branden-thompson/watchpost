import pathlib
# 0.19.5 Q5.6 (DR-13): a new hour gets a row of missing values in every
# measure kept beside the hours, so each row stays beside its hour.
# Killed by TestAnHourGoesInItsPlace.
p = pathlib.Path('domains/temperature/rain.go'); s = p.read_text()
old = '\tfor _, r := range rows { // bounded by the measures (P10-02)\n'
assert s.count(old) == 1, 'm774'
p.write_text(s.replace(old, '\tfor _, r := range rows[:1] { // bounded by the measures (P10-02)\n', 1))
