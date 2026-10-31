import pathlib
# 0.19.0 FR-4.5 (D-83): a rate header's first sighting is noted by host and name, never its value.
# Killed by TestOnlyRateHeadersAreRead.
p = pathlib.Path("domains/propagation/propagation.go"); s = p.read_text()
old = "\t\t\tf.note(\"Propagation: \" + host + \" sent \" + name)"
assert old in s, "m137"
p.write_text(s.replace(old, "\t\t\tf.note(\"Propagation: \" + host + \" sent \" + name + \" \" + v)"))
