import pathlib
# FR-4.10: a released map keeps its sessions' generations, so an old tick asks nothing.
# Killed by TestAReleasedMapDropsTheOldSessionsTick.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tpropGen: p.propGen, propTickSeq: p.propTickSeq}'
assert old in s, 'm169'
p.write_text(s.replace(old, '}'))
