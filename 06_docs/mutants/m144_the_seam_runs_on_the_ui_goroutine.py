import pathlib
# 0.19.0 FR-4.8: the update is called in its command, never in Update.
# Killed by TestTheUpdateNeverRunsOnTheUIGoroutine.
p = pathlib.Path("modes/tty/map_prop_update.go"); s = p.read_text()
old = "\tnoise := d.mapPropNoise\n\treturn d, func() tea.Msg { return propUpdatedMsg{gen: gen, res: <-seam(ctx, noise)} }"
assert old in s, "m144"
p.write_text(s.replace(old, "\tnoise := d.mapPropNoise\n\tres := seam(ctx, noise)\n\treturn d, func() tea.Msg { return propUpdatedMsg{gen: gen, res: <-res} }"))
