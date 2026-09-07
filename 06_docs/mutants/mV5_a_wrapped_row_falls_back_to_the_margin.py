import pathlib
# A row's continuation lines lose their hanging indent and fall back to the
# margin, where they read as another way out rather than as the rest of the row
# above (HUM LEAD, UAT 2026-09-05).
#
# RE-ANCHORED AT the wrap: the previous form guarded a TRUNCATION that the same
# UAT round ruled out, so the rule it measured no longer exists.
p = pathlib.Path("platform/render/text.go"); s = p.read_text()
old = "\tpad := strings.Repeat(\" \", displayWidth(head))"
new = "\tpad := \"\""
assert old in s, "mV5"
p.write_text(s.replace(old, new, 1))
