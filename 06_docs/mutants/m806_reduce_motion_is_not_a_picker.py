import pathlib
# 0.19.5 D-36: Reduce motion's row loses picker: true, so the chip row no
# longer says what its arrows change and its chip words go unread.
# Killed by TestReduceMotionsChipsNameIt and TestEveryChipWordIsRead.
p = pathlib.Path('modes/tty/setup_rows.go'); s = p.read_text()
old = 'rowReduceMotion:  {id: rowReduceMotion, group: groupUI, scope: scopeShared, kind: rowPicker, picker: true},'
assert s.count(old) == 1, 'm806'
p.write_text(s.replace(old, 'rowReduceMotion:  {id: rowReduceMotion, group: groupUI, scope: scopeShared, kind: rowPicker},', 1))
