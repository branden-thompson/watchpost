import pathlib
# 0.19.0 FR-2.2: the Broadcaster tab says the device is shared.
# Killed by TestTheSharedDeviceIsSaid.
p = pathlib.Path('modes/tty/setup_notices.go'); s = p.read_text()
old = '\t\tslots = []noticeSlot{slotOf(settingsNotice{"Audio", sharedDeviceNote, noticeInfo}, true)} // FR-2.2, D-85'
assert old in s, 'm296'
p.write_text(s.replace(old, '\t\tslots = nil // FR-2.2, D-85'))
