import pathlib
# Every run rides the lane, so the whole schedule is one serial queue: the next
# card\'s 1.03 s build waits behind the whole of this card\'s read. The ordering
# claims all still hold — this is the mutant the control test exists for.
#
# RE-ANCHORED at F-D2 round 2: the routing test is sharesAnOutput.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\t\tif !sharesAnOutput(group) {\n"
new = "\t\tif false {\n"
assert old in s, "mD1"
p.write_text(s.replace(old, new, 1))
