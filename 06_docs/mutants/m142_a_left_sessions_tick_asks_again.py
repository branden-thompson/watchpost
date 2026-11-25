import pathlib
# 0.19.0 FR-4.10: a refresh tick from a session since left asks nothing.
# Killed by TestTheModeRefreshesWhileOpen.
p = pathlib.Path("modes/tty/map_prop_update.go"); s = p.read_text()
old = 'func (mm mapModel) applyPropTick(v propTickMsg, e mapEnv) (mapModel, tea.Cmd) {\n\tif v.gen != mm.mapPane.propGen {'
assert old in s, "m142"
p.write_text(s.replace(old, 'func (mm mapModel) applyPropTick(v propTickMsg, e mapEnv) (mapModel, tea.Cmd) {\n\tif false {'))
