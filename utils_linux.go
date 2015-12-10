// +build linux darwin

package ssf

import (
	"os"
	"syscall"
)

func trylockFile(path string) error {
	fd, err := syscall.Open(path+"/.lock", syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC, 0660)
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
