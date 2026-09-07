# RE-ANCHORED AT T3.10b: CueTicker carries the card's Slot (BD-8), so a card that
# cues ITSELF line by line — a takeover — can be told from one the band shows a
# single callout for. The rule is unchanged; the line it is written against moved.
import pathlib
# The two effects swapped. The band is told to show a callout after the read has
# already started — today's accidental ordering, undone.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	return d, []Effect{
		CueTicker{ID: onAir.ID, Headline: onAir.Headline, Slot: onAir.Slot},
		Speak{ID: onAir.ID, Slot: onAir.Slot, Script: onAir.Script},
	}, false"""
new = """	return d, []Effect{
		Speak{ID: onAir.ID, Slot: onAir.Slot, Script: onAir.Script},
		CueTicker{ID: onAir.ID, Headline: onAir.Headline, Slot: onAir.Slot},
	}, false"""
assert old in s, "m90"
p.write_text(s.replace(old, new, 1))
