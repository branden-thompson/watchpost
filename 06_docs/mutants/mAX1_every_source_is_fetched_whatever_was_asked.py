import pathlib
# The deck gathers fire whatever the request asked for. The report still SAYS the
# right things — `Compose` skips a source with no data either way — so nothing on
# air changes and nothing looks wrong. What changes is that a QUAKE-only request
# pays for the fire feeds, every time, and the subset the operator chose buys
# them nothing (R2).
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = "	if want.Has(report.Fire) && d.fire != nil {"
assert old in s, "mAX1"
p.write_text(s.replace(old, "	if d.fire != nil {", 1))
