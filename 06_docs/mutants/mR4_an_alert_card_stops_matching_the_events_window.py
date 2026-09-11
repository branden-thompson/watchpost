import pathlib
# The alert card gets a palette of its own instead of the [w] window's tints, so
# the two surfaces name the same hazard in different colours.
#
# HUM LEAD, 2026-09-11: "Alert cards should be color coded to match the most
# severe alert based on the [w] category bkgs in Observer (they should match)."
#
# `_ = worst` RATHER THAN A DELETION, which is the rule mA3 cost a gate run to
# learn: a mutation that removes the only use of a name stops COMPILING rather
# than stops being true, and a mutant that cannot be applied is no evidence
# either way.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\t\treturn fg + \";\" + render.Tok(category.Of(worst).Tint)"
new = "\t\t\t_ = worst\n\t\t\treturn fg + \";\" + render.Tok(render.TickerEmergencyBG)"
assert old in s, "mR4"
p.write_text(s.replace(old, new, 1))
