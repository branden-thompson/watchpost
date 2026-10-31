import pathlib
# 0.19.0 FR-4.7 (D-113): only https to a host go-ionomaps exports.
# Killed by TestTheClientRefusesAnUnlistedHost.
p = pathlib.Path("domains/propagation/propagation.go"); s = p.read_text()
old = "\tif !f.hosts[host] {\n\t\treturn ionomaps.Response{}, errUnlisted\n\t}"
assert old in s, "m136"
p.write_text(s.replace(old, "\t_ = f.hosts"))
