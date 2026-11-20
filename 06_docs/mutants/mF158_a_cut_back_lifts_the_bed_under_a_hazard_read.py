import pathlib
# The Restore executor gives the bed back by itself, without the arbiter: it
# lets go of the rail's hold and lifts the bed at once. The Director answers
# the operator's cut-back with Restore even while a hazard is being read,
# because nothing it holds is carried any more, so this brings the programme
# back up over a Tornado Warning (F-158; 0.19.6 RM-4, design AP-7.1).
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """\t\tx.voice.releaseBed() // NOT x.mc.unhold()"""
new = """\t\tx.mc.unhold()
\t\tx.mc.takeBack() // NOT x.mc.unhold()"""
assert old in s, "mF158"
p.write_text(s.replace(old, new, 1))
