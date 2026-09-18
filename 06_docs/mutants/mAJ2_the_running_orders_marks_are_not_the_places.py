import pathlib
# The running order's prefix marks stop being the PLACE's, so a report about a
# town with a warning, a fire and a quake near it says nothing where the operator
# is choosing what to read (D-116, HUM LEAD: "Relevant Prefix Alerts").
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = """			row.Marks.Fire, row.Marks.FireHot, row.Marks.Seismic = w.Fire, w.FireHot, w.Seismic"""
assert old in s, "mAJ2"
p.write_text(s.replace(old, "", 1))
