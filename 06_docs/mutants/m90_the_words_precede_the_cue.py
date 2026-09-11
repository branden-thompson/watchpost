# RE-ANCHORED AT D-82: the cue carries the card's TRACK now, beside its slot. The
# rule is unchanged; the line it is written against has moved for the fourth time,
# which is what a re-anchor is for.
import pathlib
# The two effects swapped. The band is told to show a callout after the read has
# already started — today's accidental ordering, undone.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\treturn d, []Effect{
\t\tCueTicker{ID: onAir.ID, Headline: onAir.Headline, Slot: onAir.Slot, Track: track},
\t\tSpeak{ID: onAir.ID, Slot: onAir.Slot, Headline: onAir.Headline, Track: track, Script: onAir.Script},
\t}, false"""
new = """\treturn d, []Effect{
\t\tSpeak{ID: onAir.ID, Slot: onAir.Slot, Headline: onAir.Headline, Track: track, Script: onAir.Script},
\t\tCueTicker{ID: onAir.ID, Headline: onAir.Headline, Slot: onAir.Slot, Track: track},
\t}, false"""
assert old in s, "m90"
p.write_text(s.replace(old, new, 1))
