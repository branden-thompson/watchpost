import pathlib
# 0.19.0 FR-4.8: the update is called in its command, never in Update.
# Killed by TestTheUpdateNeverRunsOnTheUIGoroutine.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\treturn d, func() tea.Msg {\n\t\treturn propUpdatedMsg{gen: gen, res: <-seam(ctx, opts), correct: opts.CorrectNoReadings}\n\t}'
assert old in s, 'm144'
p.write_text(s.replace(old, '\tres := seam(ctx, opts)\n\treturn d, func() tea.Msg {\n\t\treturn propUpdatedMsg{gen: gen, res: <-res, correct: opts.CorrectNoReadings}\n\t}'))
