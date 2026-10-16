import pathlib
# 0.19.0 D-136: the status block, after the origin and before the targets.
# Killed by TestTheWordsFollowTheRuledOrder.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tout = append(out, d.propStatusBlock())\n'
assert old in s, 'm352'
p.write_text(s.replace(old, ''))
