import pathlib
# A declined effect that named a card is reported but the card is not failed, so the
# schedule waits at standby for words that will never come (DR-21).
#
# RE-ANCHORED AT THE BUILD-EXIT RED TEAM (I-2): the decline's Failed now carries
# Routed, and a comment sits between the guard and the return. The mutation is
# unchanged — the guard is widened so the card is never failed.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tif id == "" {\n\t\treturn nil\n\t}\n\t// A DECLINE IS ROUTED BY DEFINITION: the executor refused BY NAME and said\n\t// why, and the producer offers the alerts again. It is not the station\n\t// going quiet, which is what the fault window is for (I-2).\n\treturn []lineup.Event{lineup.Failed{ID: id, Reason: why, Routed: true}}\n'
new = '\tif id == "" || id != "" {\n\t\treturn nil\n\t}\n\t// A DECLINE IS ROUTED BY DEFINITION: the executor refused BY NAME and said\n\t// why, and the producer offers the alerts again. It is not the station\n\t// going quiet, which is what the fault window is for (I-2).\n\treturn []lineup.Event{lineup.Failed{ID: id, Reason: why, Routed: true}}\n'
assert old in s, "mC3"
p.write_text(s.replace(old, new, 1))
