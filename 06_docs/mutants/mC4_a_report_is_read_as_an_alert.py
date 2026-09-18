import pathlib
# A location report's words are read through the narrator as one aside clip — the
# wrong path, quietly, instead of going out on the broadcast engine. A chosen read
# REPLACES the bed (D-33); a narration speaks OVER it.
#
# ITS ANCHOR HAS GONE STALE TWICE, and both times for the same reason: the rule
# lived in a DECLINE, and a decline is a sentence. It is a fork now — the lane
# decides who reads (F-91) — which is a line that says what it does.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\tif v.Track != lineup.AlertRail {\n\t\treturn x.broadcast(ctx, v)\n\t}\n"
new = ""
assert old in s, "mC4"
p.write_text(s.replace(old, new, 1))
