import pathlib
# The rail cleared before a new burst is queued — bounds applied after
# admission, which is breakingCap's defect wearing a new hat: cards already
# promised a read are dropped unread.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "	held := len(d.lineup.tracks[AlertRail])"
new = "	d.lineup.tracks[AlertRail] = nil\n\theld := len(d.lineup.tracks[AlertRail])"
assert old in s, "m99"
p.write_text(s.replace(old, new, 1))
