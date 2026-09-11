import pathlib
# The `return` goes, so a main-track CARD's statuses reach the bed's half of
# onStatus as well. The engine reports the same Playing and Stopped for a card as
# it does for the bed, so the Director then reads a card's start as "the bed
# landed here, start its dwell" and a card's end as "the cycle finished, move the
# rotation on" — the station's programme driving the listener's watchlist.
#
# THE noteRead CALL STAYS, deliberately. Deleting the whole block leaves `rd`
# unused and the tree will not compile, and a mutant that cannot be applied is
# no evidence either way. What is mutated is the rule — that a read's statuses
# are the READER's and nobody else's — not the block that carries it.
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = '\tif mode == "read" {\n\t\tnoteRead(rd, st, ended)\n\t\treturn\n\t}\n'
new = '\tif mode == "read" {\n\t\tnoteRead(rd, st, ended)\n\t}\n'
assert old in s, "mCD"
p.write_text(s.replace(old, new, 1))
