import pathlib
# 0.19.0 FR-3.3: with no selected place the layers' sentences are still said.
# Killed by TestALayerAddsItsSentence.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\t\treturn d.layerWords() // no place to say'
assert old in s, 'm366'
p.write_text(s.replace(old, '\t\treturn nil // no place to say'))
