import pathlib
# The scroll rail stops being told where its window sits, so the thumb is drawn
# at the top however far the operator has scrolled — a control that says the
# same thing in every state, which is worse than no control because it looks
# like one.
#
# It was this way until D-87 and invisible, because everything fitted on one
# screen. A card is a manifest now and ten of them do not.
p = pathlib.Path("modes/tty/broadcaster_rail.go"); s = p.read_text()
old = "\t\tfor i, m := range render.Railify(make([]string, track), 1, lo,"
new = "\t\tfor i, m := range render.Railify(make([]string, track), 1, 0,"
assert old in s, "mS4"
p.write_text(s.replace(old, new, 1))
