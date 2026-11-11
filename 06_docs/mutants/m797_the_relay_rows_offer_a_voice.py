import pathlib
# 0.19.5 Q9.3 (JM-8, A-22): a picker that is not a voice names what its arrows change.
# Killed by TestEveryPickerNamesWhatItsArrowsChange and TestTheRelayRowsOfferNoVoice.
p = pathlib.Path('modes/tty/setup_layout.go'); s = p.read_text()
old = '\trowRelayDwell:      "Rotation",\n\trowRelayLang:       "Language",\n'
assert s.count(old) == 1, 'm797'
p.write_text(s.replace(old, '', 1))
