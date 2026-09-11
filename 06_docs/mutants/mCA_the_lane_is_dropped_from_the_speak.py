# RE-ANCHORED AND RE-AIMED AT F-91, and the re-aim is the point: this used to drop
# the SLOT, because the slot was what chose the reader. It no longer is — both
# lanes perform now, and WHICH ONE is the card's TRACK (a transition's slot says
# only that the Director minted it). Dropping the slot today changes nothing and
# the mutant would have SURVIVED as a rule nobody tests.
import pathlib
# BD-8 undone for the lane: the speak names no track, so every card arrives with
# Track's zero value — MainTrack — and the alert rail is read as the programme.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\tSpeak{ID: onAir.ID, Slot: onAir.Slot, Headline: onAir.Headline, Track: track, Script: onAir.Script},\n"
new = "\t\tSpeak{ID: onAir.ID, Slot: onAir.Slot, Headline: onAir.Headline, Script: onAir.Script},\n"
assert old in s, "mCA"
p.write_text(s.replace(old, new, 1))
