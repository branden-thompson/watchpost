import pathlib
# A cast save from Observer's Settings hard-recasts whatever is running. On the
# console that is the card on the air, so the station changes voice mid-sentence
# because the operator looked at a settings row. "The listener is waiting to hear
# it" is true of the OPERATOR on Observer; on the console the listener is the
# AUDIENCE (D-91).
p = pathlib.Path("app/cast.go"); s = p.read_text()
old = "\tif src != nil && d.monitorHasTheAir() {"
new = "\tif src != nil {"
assert old in s, "mZ2"
p.write_text(s.replace(old, new, 1))
