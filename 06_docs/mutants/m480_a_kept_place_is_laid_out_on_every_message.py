import pathlib
# 0.19.0 BUILD red team round 2 (D-211, PN2-5): a message that changes nothing lays out no words.
# Killed by TestAKeptPlaceLaysOutNoWordsWhileNothingChanges.
p = pathlib.Path('modes/tty/map_prop_place.go'); s = p.read_text()
old = 'anchor != d.mapPane.wordsAnchored {'
assert old in s, 'm480'
p.write_text(s.replace(old, 'anchor == anchor {', 1))
