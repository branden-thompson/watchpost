import pathlib
# A rail card reads in the [space] read's class: the visualizer follows it and it
# queues behind a takeover instead of being one.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tx.voice.Run(ctx, narrateBreaking, cast.Breaking, x.audible(), func(ctx context.Context, s *speaker) {\n'
new = '\tx.voice.Run(ctx, narrateRead, cast.Breaking, x.audible(), func(ctx context.Context, s *speaker) {\n'
assert old in s, "mC9"
p.write_text(s.replace(old, new, 1))
