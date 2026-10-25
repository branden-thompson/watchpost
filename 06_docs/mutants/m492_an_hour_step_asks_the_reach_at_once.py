import pathlib
# 0.19.0 BUILD red team round 2 (D-227): hour steps settle as pans do before the reach is asked.
# Killed by TestHourStepsAskOneReachAtATime.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = 'func (d Dashboard) armPropReach() (Dashboard, tea.Cmd) {\n\tif d.viewSettling() {'
assert s.count(old) == 1, 'm492'
p.write_text(s.replace(old, 'func (d Dashboard) armPropReach() (Dashboard, tea.Cmd) {\n\tif false && d.viewSettling() {', 1))
