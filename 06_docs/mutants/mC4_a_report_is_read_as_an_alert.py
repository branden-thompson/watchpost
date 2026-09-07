import pathlib
# A location report's words are read through the narrator as one aside clip — the
# wrong path, quietly, instead of being declined until T3.2 brings the right one.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tif !onTheRail(v.Slot) {\n\t\treturn x.decline(v, v.ID, "read by the main track, which arrives with T3.2")\n\t}\n'
new = '\tif !onTheRail(v.Slot) && v.Slot < 0 {\n\t\treturn x.decline(v, v.ID, "read by the main track, which arrives with T3.2")\n\t}\n'
assert old in s, "mC4"
p.write_text(s.replace(old, new, 1))
