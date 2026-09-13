import pathlib
# The takeover's three-cell gutter goes back into the fill column, so a hazard
# that uses its last cell sits flush against the place name: "SEVERE THUNDERSTORM
# WARN…Harper, KS" is what the HUM LEAD saw (D-103, the column spec).
p = pathlib.Path("platform/render/alert_table.go"); s = p.read_text()
old = """		{Name: "kind", Header: "ALERT TYPE", Width: kind + alertGap2, Alignment: "left"},"""
assert old in s, "mAB3"
p.write_text(s.replace(old, """		{Name: "kind", Header: "ALERT TYPE", Width: kind, Alignment: "left"},""", 1))
