import pathlib
# The standby notice reaches UP NEXT as well as LIVE, so a station that has just
# opened reports "no reports read or active" about the very slot the Composer is
# working on. D-84 puts the Composer on UP NEXT precisely so the line is ready at
# SHIFT+ENTER; the WAITING placeholder is what says that work is happening.
#
# RE-ANCHORED AT D-110. The rule used to live in a loop over read REGIONS; the
# regions retired with the card column and there are two read slots left — the
# air box's LIVE row and the UP NEXT card — so the mutation is one slot along.
p = pathlib.Path("modes/tty/broadcaster_upnext.go"); s = p.read_text()
old = "\t\tc = lineup.Card{Headline: b.waiting(o)}"
assert old in s, "mX3"
p.write_text(s.replace(old, "\t\tc = lineup.Card{Headline: bcStandbyNotice}", 1))
