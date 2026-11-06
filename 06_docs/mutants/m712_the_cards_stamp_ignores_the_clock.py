import pathlib
# 0.19.5 D-29: the card's own stamp is in the clock Setting.
# Killed by TestTheCardStampsFollowTheClock.
p = pathlib.Path('modes/tty/broadcaster_manifest.go'); s = p.read_text()
old = 'c.BuiltAt.Format("01/02/06  ") + o.Clock.TimeSec(c.BuiltAt)'
assert s.count(old) == 1, 'm712'
p.write_text(s.replace(old, 'c.BuiltAt.Format("01/02/06  15:04:05")', 1))
