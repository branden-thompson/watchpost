import pathlib
# 0.19.5 Q5.6 (DR-12): an hour with no readings is NaN throughout, never 0.
# Killed by TestAMissingRowIsAllMissing.
p = pathlib.Path('app/history.go'); s = p.read_text()
old = '\tfor i := range out { // P10-02\n\t\tout[i] = math.NaN()\n'
assert s.count(old) == 1, 'm773'
p.write_text(s.replace(old, '\tfor i := range out { // P10-02\n\t\tout[i] = math.Copysign(0, 1)\n', 1))
