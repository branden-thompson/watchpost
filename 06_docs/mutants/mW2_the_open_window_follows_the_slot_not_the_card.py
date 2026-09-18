import pathlib
# The open card window refreshes by SLOT POSITION instead of by the card's ID, so a
# card promoted while the operator is reading it silently swaps the report in the
# window for whichever card took its place. `Card.ID` exists precisely because a
# position is not an identity: "ID addresses the card for the life of the lineup."
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "		if id, rows, ok := r.broadcaster.cardDetail(i); ok && id == r.observer.cardID {"
new = "		if id, rows, ok := r.broadcaster.cardDetail(i); ok && i == 1 {"
assert old in s, "mW2"
p.write_text(s.replace(old, new, 1))
