import pathlib
# A voice that DIED mid-read comes home as if the operator had stopped it. The
# seam grades a player failure as a fault and a stop as routed (F-150); collapse
# the two and a dead voice is a deliberate non-delivery, which never counts
# toward the run and never reaches the operator's band.
p = pathlib.Path("app/mainread.go"); s = p.read_text()
old = "\tcase st.State == player.Failed:\n\t\tr.finish(errors.New(\"the player failed: \" + st.Err))"
new = "\tcase st.State == player.Failed:\n\t\tr.finish(errReadStopped)"
assert old in s, "mCL4"
p.write_text(s.replace(old, new, 1))
