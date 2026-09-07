import pathlib
# The Settings choice is stored and the Director is never told.
#
# The Director HOLDS the dwell it was last given, so storing the new one without
# re-sending it leaves the station rotating on the old number until something
# else happens to re-send it — up to five minutes away on a station that is
# already rotating. From the listener's chair that is "the setting does not
# work", which is the complaint the setting exists to answer.
p = pathlib.Path("app/dashboard.go"); s = p.read_text()
old = "\t\tlp.deck.SetRepeat(mode, queue)"
new = "\t\t_, _ = mode, queue"
assert old in s, "mM5"
p.write_text(s.replace(old, new, 1))
