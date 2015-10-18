package ssf

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var serviceCmd *exec.Cmd

func KillService() {
	if nil == serviceCmd {
		return
	}
	serviceCmd.Process.Kill()
}

func RestartService(procName string) error {
	if nil != serviceCmd {
		KillService()
	}
	path, err := filepath.Abs(os.Args[0])
	if nil != err {
		return err
	}
	dir, _ = filepath.Split(path)
	serviceCmd = exec.Command(dir+"/"+procName, "5")
	err = serviceCmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
}
