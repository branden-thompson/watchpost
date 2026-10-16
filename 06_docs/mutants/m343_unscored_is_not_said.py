import pathlib
# 0.19.0 D-198: a target's unscored station, to verify on air.
# Killed by TestEachTargetSaysItsStation.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif n.Known && n.Unscored {\n\t\tw += ", unscored: verify on air"\n\t}\n\treturn w\n}\n\n// spokenRules'
assert old in s, 'm343'
p.write_text(s.replace(old, '\treturn w\n}\n\n// spokenRules'))
