import pathlib
# 0.19.5 D-29: the card window's stamp is in the clock Setting.
# Killed by TestTheCardStampsFollowTheClock.
p = pathlib.Path('modes/tty/broadcaster_manifest.go'); s = p.read_text()
old = 'at.Format("Monday January 2, 2006 @ ") + o.Clock.TimeSec(at)'
assert s.count(old) == 1, 'm711'
p.write_text(s.replace(old, 'at.Format("Monday January 2, 2006 @ 15:04:05")', 1))
