import pathlib
# The data stamp reads the console's raw `now` FIELD instead of its clock, and
# production never sets that field — so the AGE is missing in the real app and
# present in every test that injects a time. Two carriers of one fact, with the
# tests holding the one that works.
#
# The HUM LEAD saw it in UAT: "DATA PULLED: Friday September 11, 2026 @
# 17:44:56" with no "(2 MIN AGO)" after it.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "cardPulled(o, c, b.clock,"
new = "cardPulled(o, c, b.now,"
assert old in s, "mT5"
p.write_text(s.replace(old, new, 1))
