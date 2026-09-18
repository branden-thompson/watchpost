import pathlib
# The console's own controls stop standing aside for a window composited over
# them, so `←`/`→` step the bed's relay and `↑`/`↓` scroll the queue while the
# operator is moving through the status or diagnostics window.
#
# THE BED'S ARROWS HAD THIS SINCE D-78 and nothing saw it, because no gate drove
# those keys with a window up. The queue's arrows are the ones the modal
# reachability gate drives — "a line the keyboard cannot bring on screen is not
# in the window" — so adding them is what surfaced a bug two releases old.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "\treturn r.active == SurfaceBroadcaster && !r.observer.ModalOpen()"
new = "\treturn r.active == SurfaceBroadcaster"
assert old in s, "mS7"
p.write_text(s.replace(old, new, 1))
