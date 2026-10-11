import pathlib
# FR-4.10 (D-94): only the latest refresh tick asks; U starts no second chain.
# Killed by TestOnlyTheLatestRefreshTickAsks.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif v.seq != d.mapPane.propTickSeq {\n\t\treturn d, nil // a later tick is set\n\t}\n'
assert old in s, 'm168'
p.write_text(s.replace(old, ''))
