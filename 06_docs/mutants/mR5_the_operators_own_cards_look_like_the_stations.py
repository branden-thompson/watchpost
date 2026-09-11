import pathlib
# Origin stops changing the ground, so a card the OPERATOR asked for is
# indistinguishable from one the station proposed for itself — the one thing the
# narrow ruling asked the colour to say.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tif c.Origin == lineup.FromOperator {\n\t\treturn fg + \";\" + render.Tok(render.CardOperatorBG)\n\t}\n"
assert old in s, "mR5"
p.write_text(s.replace(old, "", 1))
