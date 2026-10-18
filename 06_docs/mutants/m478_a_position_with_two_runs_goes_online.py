import pathlib
# 0.19.0 BUILD red team round 2 (FR-4.9, PN2-4): a query with two runs of figures is refused as a position.
# Killed by TestAPositionInAnotherFormNeverGoesOnline.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = 'len(numberGroup.FindAllString(q, 2)) >= 2'
assert old in s, 'm478'
p.write_text(s.replace(old, 'len(numberGroup.FindAllString(q, 3)) >= 3', 1))
