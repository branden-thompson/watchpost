import pathlib
# 0.19.0 FR-4.9: no error quotes what was typed.
# Killed by TestNoLookupErrorQuotesTheQuery.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = 'errors.New("no match in the offline index'
assert old in s, 'm318'
p.write_text(s.replace(old, 'errors.New("no match for " + q + " in the offline index'))
