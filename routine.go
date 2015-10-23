package ssf

import (
	"github.com/golang/glog"
	"os"
	"time"
)

func checkPpid() {
	if os.Getppid() == 1 {
		glog.Errorf("Exit current process since parent exited.")
		os.Exit(1)
	}
}

func initRoutine() {
	hbTickChan := time.NewTicker(time.Millisecond * 1000).C
	checkTickChan := time.NewTicker(time.Millisecond * 5000).C
	go func() {
		for {
			select {
			case <-hbTickChan:
				ssfClient.checkPartitionConns()
				ssfClient.heartbeat()
			case <-checkTickChan:
				checkPpid()
				ssfClient.checkHeartbeatTimeout()
				ssfClient.replayWals()
			}
		}
	}()
}
