import pathlib
# settle treats a listener's pause like a takeover's suspension, so a paused
# read starts speaking again the moment an unrelated alert finishes — the hold
# silently undone by something the listener had nothing to do with.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = "	if s := d.innermostResumable(); s != nil"
new = "	if s := d.innermostSuspended(); s != nil"
assert old in s, "mJ1"
p.write_text(s.replace(old, new, 1))
