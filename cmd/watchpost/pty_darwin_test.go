package main

// pty_darwin_test.go — a pseudo-terminal pair on macOS, from the standard
// library alone (D-271): /dev/ptmx, then grant, unlock and name the secondary.

import (
	"bytes"
	"os"
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
	var name [128]byte
	// The pointer is converted in the call itself, the only form the runtime
	// keeps valid across a system call.
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCPTYGRANT, 0); e != 0 {
		return failPTY(p, e)
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCPTYUNLK, 0); e != 0 {
		return failPTY(p, e)
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); e != 0 {
		return failPTY(p, e)
	}
	end := bytes.IndexByte(name[:], 0)
	if end < 0 {
		end = len(name)
	}
	s, err := os.OpenFile(string(name[:end]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return failPTY(p, err)
	}
	return p, s, nil
}
