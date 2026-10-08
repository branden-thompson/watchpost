import pathlib
# #27 (0.19.0 W1.2): a day goes to the month's last part only while that part stays within the budget.
# Killed by TestARollUpNeverPassesTheReadCap.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "	if lastPath != \"\" && fitsBudget(withDay(last.Days, entry)) {"
assert old in s, "m111"
p.write_text(s.replace(old, "	if lastPath != \"\" {"))
