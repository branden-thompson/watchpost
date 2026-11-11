import pathlib
# 0.19.5 Q2.11 (RN-10): while the launch phase lasts, the publisher's window is
# the launch window, read off the publisher's own clock.
# Killed by TestFirstPublishIsImmediateAndLaunchFillsProgressively.
p = pathlib.Path('app/pipelines.go'); s = p.read_text()
old = '\tif pb.launchWindow > 0 && now().Before(pb.launchUntil) {\n'
assert s.count(old) == 1, 'm716'
p.write_text(s.replace(old, '\tif pb.launchWindow > 0 && now().Add(time.Hour).Before(pb.launchUntil) {\n', 1))
