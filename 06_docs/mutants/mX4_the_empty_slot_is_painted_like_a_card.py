import pathlib
# The empty LIVE slot is painted on the CARD ground instead of its own grey, so a
# slot with nothing in it reads as a card the operator cannot make out — which is
# the opposite of what an empty state is for. The HUM LEAD asked for "a grey box",
# and grey is the one colour that claims neither a card nor a state.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\treturn render.Tok(render.CardText) + \";\" + render.Tok(render.CardEmptyBG)"
new = "\treturn render.Tok(render.CardText) + \";\" + render.Tok(render.CardBG)"
assert old in s, "mX4"
p.write_text(s.replace(old, new, 1))
