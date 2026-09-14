import pathlib
# The pool lookup stops checking and admits whatever was typed as a pooled
# location. The window then accepts a place the station cannot broadcast about,
# and the request is scheduled for a location with no data behind it — the one
# outcome ruling 2 exists to prevent.
#
# THE POOL IS THE WHOLE ANSWER now that the resolver fall-through is gone (HUM
# LEAD, 2026-09-14: "either what they type is a valid location within the service
# radius or not"), which makes this check the only thing between a typo and a
# scheduled card.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """		if matchesQuery(ref, q) {"""
assert old in s, "mBC3"
p.write_text(s.replace(old, """		if true {""", 1))
