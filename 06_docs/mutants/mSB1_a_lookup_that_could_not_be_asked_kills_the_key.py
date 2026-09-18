import pathlib
# D-157. The request window forgets the retry arm, so a lookup that could not be
# MADE — a timeout, a cancelled context — leaves enter doing nothing. The helper
# line, shared and therefore right, still reads "The lookup did not answer; press
# enter to try again". A window instructing an action the window refuses.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = "\t\tcase submitAsk, submitRetry:"
assert old in s, "mSB1"
p.write_text(s.replace(old, "\t\tcase submitAsk:", 1))
