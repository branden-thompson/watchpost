import pathlib
# 0.19.5 Q5.6 (DR-8): an id is bounded at maxIDRunes runes.
# Killed by TestAnIDIsBoundedInRunes.
p = pathlib.Path('domains/globalfeed/event.go'); s = p.read_text()
old = 'return plaintext.ClampRunes(s, maxIDRunes) }'
assert s.count(old) == 1, 'm768'
p.write_text(s.replace(old, 'return plaintext.ClampRunes(s, maxIDRunes-1) }', 1))
