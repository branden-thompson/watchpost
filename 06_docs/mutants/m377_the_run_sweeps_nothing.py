import pathlib
# 0.19.0 FR-9.2: the run's start sweeps the orphans.
# Killed by TestTheRunSweepsOrphansAndEndsItsChildren.
p = pathlib.Path('app/saychildren.go'); s = p.read_text()
old = '\t\tswept, failed = synth.SweepOrphans(dir, procs, os.Getuid(), time.Now())\n'
assert old in s, 'm377'
p.write_text(s.replace(old, '\t\tswept, failed = 0*os.Getuid(), time.Now().Second()*0\n'))
