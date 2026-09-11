import pathlib
# The cross-check goes, so a rail card that arrives with no track — Track's zero
# value is MainTrack, so anything built without one — is played on the broadcast
# engine. Its attention tone never sounds and its per-alert callouts never go up:
# a tornado warning, delivered as the weather.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tif onTheRail(v.Slot) {\n\t\treturn x.decline(v, v.ID, "a rail card reached the programme\'s reader: its tone and its callouts would be lost")\n\t}\n'
new = ""
assert old in s, "mCB"
p.write_text(s.replace(old, new, 1))
