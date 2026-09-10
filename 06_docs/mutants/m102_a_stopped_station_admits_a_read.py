# 0.16.0 P3, DR-3. Admit a rotation card while the programme is stopped.
#
# ADMISSION IS A PROMISE TO READ, so a track that cannot advance must not accept
# one: a stopped programme would otherwise pile up a rotation nobody can drop,
# and every card of it would be owed a read the moment the listener pressed
# start.
import pathlib
p = pathlib.Path("platform/lineup/rotation.go"); s = p.read_text()
old = "\tif !d.advances(MainTrack) {\n"
assert old in s, "m102"
p.write_text(s.replace(old, "\tif false && !d.advances(MainTrack) {\n"))
