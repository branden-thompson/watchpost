import pathlib
# 0.19.0 FR-1.10: a query nothing local matches is answered online.
# Killed by TestALocalMatchSendsNothing.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\tworld, ok := r.fallback.(WorldFallback)\n'
assert old in s, 'm384'
p.write_text(s.replace(old, '\tworld, ok := r.fallback.(WorldFallback)\n\tok = ok && false\n'))
