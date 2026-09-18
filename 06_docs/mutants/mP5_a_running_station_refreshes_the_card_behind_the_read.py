import pathlib
# The refresh stops being bounded to a station that cannot advance, so a RUNNING
# station re-asks for the words of the card waiting behind the one on the air —
# spending the Composer mid-broadcast on a card that is about to be replaced
# anyway. PD-3's drop is the right answer there: mid-broadcast there is no time
# to rebuild, which is the whole reason readInstead exists.
#
# ITS FIRST TEST COULD NOT FAIL and this plant said so: one card on a running
# station is READ immediately, so there was no standing-by card left to refresh.
# The test now puts one on the air and one behind it.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
old = "\tif d.advances(MainTrack) {\n\t\treturn d, nil\n\t}\n"
assert old in s, "mP5"
p.write_text(s.replace(old, "", 1))
