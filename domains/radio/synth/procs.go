package synth

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SystemProcs is the machine's process table, read through ps (FR-9.2).
// Where there is no ps - Windows, which has no say either - it finds no
// process and so ends none.
type SystemProcs struct{}

// psStart is how ps writes a process's start time (lstart), in local time.
const psStart = "Mon Jan _2 15:04:05 2006"

// Inspect reads a process's owner, parent, start time and arguments; false
// when it is gone or ps cannot say.
func (SystemProcs) Inspect(pid int) (Proc, bool) {
	out, err := exec.Command("ps", "-ww", "-o", "uid=,ppid=,lstart=,command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return Proc{}, false
	}
	return parsePS(strings.TrimSpace(string(out)))
}

// parsePS reads one ps line: uid, ppid, the five words of lstart, then the
// command and its arguments.
func parsePS(line string) (Proc, bool) {
	f := strings.Fields(line)
	if len(f) < 8 {
		return Proc{}, false
	}
	uid, errU := strconv.Atoi(f[0])
	ppid, errP := strconv.Atoi(f[1])
	started, errS := time.ParseInLocation(psStart, strings.Join(f[2:7], " "), time.Local)
	if errU != nil || errP != nil || errS != nil {
		return Proc{}, false
	}
	return Proc{UID: uid, PPID: ppid, Started: started, Args: strings.Join(f[7:], " ")}, true
}

// End kills a process.
func (SystemProcs) End(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
