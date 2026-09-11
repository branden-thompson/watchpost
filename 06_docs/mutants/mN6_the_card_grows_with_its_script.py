import pathlib
# The window returns as many rows as the script has, so a card changes HEIGHT
# when its words land — and every card below it moves at the moment the operator
# is reading one, or typing a slot number at one.
#
# readBody has stated the rule since D-68 ("the height is the same whether there
# is a script or not"); the window is the first thing that could break it.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\tfor i := range bcReadLines { // bounded by the card's height (P10-02)\n\t\tif i >= len(wrapped) {\n\t\t\trows = append(rows, \"\")\n\t\t\tcontinue\n\t\t}\n"
new = "\tfor i := range wrapped { // bounded by the script (P10-02)\n"
assert old in s, "mN6"
p.write_text(s.replace(old, new, 1))
