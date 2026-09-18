import pathlib
# D-133. The masthead paints both editions in the same word colour again, so the
# ONE place an operator can be sure which surface they are on distinguishes them
# by the WORD alone — and ctrl+o / ctrl+b swap the surfaces in place, under an
# identical wordmark, identical ladders and an identical stamp.
#
# THE TOKEN SURVIVES THE MUTATION, which is what makes it a real regression
# rather than a compile error: TitleEditionBroadcaster is still declared, still
# registered for AA, still carries each theme's orange — it is simply not the one
# the painter reaches for.
p = pathlib.Path("platform/render/sgr.go"); s = p.read_text()
old = """	if edition == EditionBroadcaster {
		return TitleEditionBroadcaster
	}
	return TitleEdition"""
new = """	return TitleEdition"""
assert old in s, "mCF1"
p.write_text(s.replace(old, new, 1))
