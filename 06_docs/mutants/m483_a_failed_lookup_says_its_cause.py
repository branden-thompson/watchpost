import pathlib
# 0.19.0 BUILD red team round 2 (D-124, PN2-9): a failed lookup is said in the listener's words, its cause kept apart.
# Killed by TestTheWorldsFailuresSayItsOwnFix.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = '\treturn "the online lookup did not answer; coordinates, a grid locator and the world\'s larger cities are found without it"'
assert old in s, 'm483'
p.write_text(s.replace(old, '\treturn "the online lookup did not answer (" + e.Cause.Error() + "); coordinates, a grid locator and the world\'s larger cities are found without it"', 1))
