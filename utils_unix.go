// +build darwin dragonfly freebsd linux openbsd solaris

package ssf

import (
	"os"
	"syscall"
)

func lockFile(file string) error {
	fd, err := syscall.Open(file, syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC, 0660)
	if err != nil {
		return err
	}
	lock := syscall.Flock_t{
		Start:  0,
		Len:    0,
		Type:   syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
	}
	return syscall.FcntlFlock(uintptr(fd), syscall.F_SETLK, &lock)
}

func trylockDir(path string) error {
	return lockFile(path + "/.lock")
}

func trylockFile(file string) error {
	return lockFile(file + ".lock")
}
