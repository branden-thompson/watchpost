import pathlib
# The rail gives the bed back without asking whether anything is still speaking,
# so a read in progress is left talking over a broadcast at full volume — the
# broadcast surging back up mid-sentence.
#
# RE-ANCHORED: the decision moved under the arbiter\'s lock, because asking
# outside it and acting on the answer was the defect a fresh review found. This
# now deletes the deferral to settle, which is where the decision lives.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = """\td.mc.unhold()
\td.settle()"""
new = """\td.mc.unhold()
\td.mc.takeBack()"""
assert old in s, "mE6"
p.write_text(s.replace(old, new, 1))
