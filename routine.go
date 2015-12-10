package ssf

import "time"

func initRoutine() {
	hbTickChan := time.NewTicker(time.Millisecond * 1000).C
	checkTickChan := time.NewTicker(time.Millisecond * 5000).C
	logTickChan := time.NewTicker(time.Millisecond * 60000).C
	go func() {
		for ssfRunning {
			select {
			case <-hbTickChan:
				ssfClient.checkPartitionConns()
				ssfClient.heartbeat()
			case <-checkTickChan:
				ssfClient.checkHeartbeatTimeout()
				ssfClient.replayWals()
			case <-logTickChan:
				ssfClient.printWalSizes()
			}
		}
	}()
}
