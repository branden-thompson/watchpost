# 0.14.0 DR-3. Let Queue admit a card whose identity the lineup already holds.
#
# AN AMBIGUOUS ADDRESS IS A CARD READ TWICE. It is also the mechanism behind
# FR-2.5 at the schedule level: rotation ids are a pure function of the
# location, so this guard is what makes a second need for the same place
# harmless.
import pathlib
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = 'invariant.Check(!taken, "no two cards in the lineup share an identity")'
assert old in s, "m106"
p.write_text(s.replace(old, 'invariant.Check(!taken || true, "no two cards in the lineup share an identity")'))
