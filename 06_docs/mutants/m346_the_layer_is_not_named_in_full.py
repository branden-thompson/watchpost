import pathlib
# 0.19.0 FR-3.2: the layer's full name, said once.
# Killed by TestPropagationWordsCarryUnitsForSpeech.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '"The Propagation mode, over the whole world: "+propLayerFull[d.mapPane.propLayer]+", "+'
assert old in s, 'm346'
p.write_text(s.replace(old, '"The Propagation mode, over the whole world: "+'))
