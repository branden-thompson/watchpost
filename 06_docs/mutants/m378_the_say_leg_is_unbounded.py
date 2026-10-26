import pathlib
# 0.19.0 FR-9.3: make test-say has a bound of its own.
# Killed by TestTheSayLegIsBoundedAndDiagnosed.
p = pathlib.Path('Makefile'); s = p.read_text()
old = 'go test -count=1 -v -timeout 6m -run'
assert old in s, 'm378'
p.write_text(s.replace(old, 'go test -count=1 -v -run'))
