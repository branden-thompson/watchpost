import pathlib
# 0.19.0 FR-2.2: the console's Help says the device is shared.
# Killed by TestTheSharedDeviceIsSaid.
p = pathlib.Path('modes/tty/help_about.go'); s = p.read_text()
old = '\tif d.surface == SurfaceBroadcaster {\n\t\tlines = append(lines, "", " "+sharedDeviceNote)'
assert old in s, 'm295'
p.write_text(s.replace(old, '\tif false {\n\t\tlines = append(lines, "", " "+sharedDeviceNote)'))
