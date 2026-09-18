import pathlib
# The card window's controls go back BELOW the keymap, where `enter` is bound to
# `actQueueOpen` — so that case closes the card window and the move is never sent.
# The operator types a position, presses enter, both windows close and the running
# order does not move (D-121, HUM LEAD: "Table does not update / redraw").
#
# THE RULE IS "AN OPEN QUESTION OWNS THE KEYBOARD" and its PLACEMENT is what
# enforces it: stated below the keymap it is not a rule, it is a comment.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
head = "	// THE CARD WINDOW'S OWN CONTROLS, BEFORE THE KEYMAP"
i = s.index(head)
j = s.index("	// A SWAP REQUEST GOES THROUGH canSwap AND NOWHERE ELSE.")
block = s[i:j]
s = s[:i] + s[j:]
anchor = "	// THE WINDOW ON TOP OWNS THE KEYS (D-58). While the diagnostics window is"
assert anchor in s, "mAO1"
p.write_text(s.replace(anchor, block + anchor, 1))
