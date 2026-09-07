import pathlib
# The mute check leaves speak, so a burst admitted before [M] lands is read
# INAUDIBLY: cued, marked as read aloud, and finished — a hazard consumed in
# silence, which is MVS-D-78's defect exactly.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\tif x.muted() {\n\t\treturn x.decline(v, v.ID, \"the listener is muted; the alerts stay new and will be offered again\")\n\t}\n"
assert old in s, "mU2"
p.write_text(s.replace(old, "", 1))
