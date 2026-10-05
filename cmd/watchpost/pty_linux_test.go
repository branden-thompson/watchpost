package main

// pty_linux_test.go — a pseudo-terminal pair on Linux, from the standard
// library alone (D-271): /dev/ptmx, then unlock and number the secondary.

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// openPTY returns the primary and secondary ends of a new pseudo-terminal.
func openPTY() (primary, secondary *os.File, err error) {
	p, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := p.Fd()
	var unlock int32
	var n uint32
	// Each pointer is converted in the call itself, the only form the runtime
	// keeps valid across a system call.
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); e != 0 {
		return failPTY(p, e)
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); e != 0 {
		return failPTY(p, e)
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.FormatUint(uint64(n), 10), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return failPTY(p, err)
	}
	return p, s, nil
}
