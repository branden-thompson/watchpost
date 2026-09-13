import pathlib
# NewRouter stops copying the operator's threshold, so the console falls back to
# the default and the [fire] bold_frp_mw setting is honoured on the watchlist and
# ignored on the console. One place reads bold on one surface and plain on the
# other — the two-carriers defect the old `bcFireBoldMW` constant WAS.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "	b.fireBoldMW = o.fireBoldMW()"
assert old in s, "mAQ1"
p.write_text(s.replace(old, "", 1))
