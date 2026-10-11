import pathlib
# IS-1: the propagation client says an address that got no reply redacted, as the data client does.
# Killed by TestThePropagationClientsErrorsCarryNoSecret.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\t\treturn Answer{}, &ReachError{URL: safe, Attempts: 1, Err: redactErr(err)}'
assert old in s, 'm176'
p.write_text(s.replace(old, '\t\treturn Answer{}, &ReachError{URL: safe, Attempts: 1, Err: err}'))
