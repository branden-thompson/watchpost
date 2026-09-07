import pathlib
# The freshness rule applied to the wrong category — a plausible slip, and one
# no count or total would notice. Stale WARNINGS get demoted and every disaster
# leads unconditionally.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = "	if c != category.Disasters {"
new = "	if c != category.Warnings {"
assert old in s, "m78"
p.write_text(s.replace(old, new, 1))
