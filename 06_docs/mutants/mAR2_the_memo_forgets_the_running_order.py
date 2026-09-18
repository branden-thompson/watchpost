import pathlib
# The key drops the line-up's generation. The Director moves a card, the message
# arrives, the model holds the new running order — and the table goes on drawing
# the old one for as long as nothing else changes. The UI lies, which is the one
# thing FR-3.3 says it must never do.
p = pathlib.Path("modes/tty/broadcaster_memo.go"); s = p.read_text()
old = "lineupGen: b.lineupGen, "
assert old in s, "mAR2"
p.write_text(s.replace(old, "", 1))
