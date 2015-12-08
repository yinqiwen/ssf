package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/edsrzf/mmap-go"
	"github.com/golang/glog"
)

const WALMetaSize int64 = 4096

type WALMeta struct {
	sequenceId   uint64
	readedOffset int64
	fileSize     int64
}

type WAL struct {
	nodeId     int32
	meta       WALMeta
	logPath    string
	log        *os.File
	mapedMeta  mmap.MMap
	logLock    sync.Mutex
	replayLock sync.Mutex
	replaying  bool
	discard    bool
}

func (wal *WAL) syncMeta() {
	buf := bytes.NewBuffer(wal.mapedMeta)
	binary.Write(buf, binary.LittleEndian, wal.meta.sequenceId)
	binary.Write(buf, binary.LittleEndian, wal.meta.readedOffset)
	binary.Write(buf, binary.LittleEndian, wal.meta.fileSize)
}

func (wal *WAL) readMeta() {
	buf := bytes.NewReader(wal.mapedMeta)
	binary.Read(buf, binary.LittleEndian, &wal.meta.sequenceId)
	binary.Read(buf, binary.LittleEndian, &wal.meta.readedOffset)
	binary.Read(buf, binary.LittleEndian, &wal.meta.fileSize)
}

func (wal *WAL) Write(content []byte) (int, error) {
	if wal.discard {
		return 0, nil
	}
	wal.logLock.Lock()
	defer wal.logLock.Unlock()
	n, err := wal.log.Write(content)
	if nil != err {
		return n, err
	}
	wal.meta.fileSize += int64(n)
	wal.syncMeta()
	return n, nil
}

func (wal *WAL) empty() bool {
	if wal.discard {
		return true
	}
	return wal.meta.readedOffset >= wal.meta.fileSize
}

func (wal *WAL) cachedDataSize() int64 {
	return wal.meta.fileSize - wal.meta.readedOffset
}

func (wal *WAL) nextSequenceId() uint64 {
	if wal.discard {
		return 0
	}
	return 0
}
func (wal *WAL) replay(writer io.Writer) {
	if wal.discard {
		return
	}
	if wal.replaying {
		return
	}
	wal.replayLock.Lock()
	wal.replaying = true
	defer func() {
		wal.replaying = false
		wal.replayLock.Unlock()
	}()

	haveDataReplay := true
	for haveDataReplay {
		wal.logLock.Lock()
		if wal.meta.readedOffset == wal.meta.fileSize {
			haveDataReplay = false
		}
		readedOffset := wal.meta.readedOffset
		fileSize := wal.meta.fileSize
		wal.logLock.Unlock()
		glog.Infof("Start replay wal for virtual node:%d with data size [%d:%d]", wal.nodeId, readedOffset, fileSize)
		if readedOffset < fileSize {
			f, err := os.Open(wal.logPath)
			if nil != err {
				return
			}
			f.Seek(WALMetaSize+wal.meta.readedOffset, 0)
			replayBuf := make([]byte, 0)
			b := make([]byte, 8192)
			for readedOffset < fileSize {
				n, err := f.Read(b)
				if n <= 0 {
					break
				}
				replayBuf = append(replayBuf, b[0:n]...)
				cursor := consumeEvents(replayBuf)
				if cursor > 0 {
					wbuf := make([]byte, cursor)
					copy(wbuf, replayBuf[0:cursor])
					replayBuf = replayBuf[cursor:]
					_, err = writer.Write(wbuf)
					if nil != err {
						glog.Warningf("Write replay log to node:%d failed:%v", wal.nodeId, err)
						haveDataReplay = false
						break
					}
					wal.meta.readedOffset += int64(cursor)
					readedOffset += int64(cursor)
				} else {
					if readedOffset == wal.meta.readedOffset {
						glog.Errorf("No event consumed in this replay for wal:%d", wal.nodeId)
					} else {
						glog.Warningf("No one complete event found in buffer with %d bytes", len(replayBuf))
					}
				}
			}
			f.Close()
			glog.Infof("Stop replay wal for virtual node:%d with current data size [%d:%d]", wal.nodeId, wal.meta.readedOffset, wal.meta.fileSize)
			wal.logLock.Lock()
			if wal.meta.readedOffset == wal.meta.fileSize {
				err = wal.log.Truncate(WALMetaSize)
				if err == nil {
					wal.meta.readedOffset = 0
					wal.meta.fileSize = 0
					wal.log.Seek(0, os.SEEK_END)
					glog.Infof("Clear wal for virtual node:%d", wal.nodeId)
				} else {
					glog.Errorf("Failed to truncate wal:%d for reason:%v", wal.nodeId, err)
				}
			}
			wal.syncMeta()
			wal.logLock.Unlock()
		}
	}

}

func newWAL(nodeId int32) (*WAL, error) {
	wal := new(WAL)
	wal.nodeId = nodeId
	wal.logPath = fmt.Sprintf("%s/wal/node_%d.wal", ssfCfg.ProcHome, nodeId)
	os.MkdirAll(ssfCfg.ProcHome+"/wal", 0770)
	var err error
	wal.log, err = os.OpenFile(wal.logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if nil != err {
		return wal, err
	}
	fstat, _ := wal.log.Stat()
	if fstat.Size() < WALMetaSize {
		err = wal.log.Truncate(WALMetaSize)
		if nil != err {
			return wal, err
		}
	}
	wal.log.Seek(0, os.SEEK_SET)
	wal.mapedMeta, err = mmap.MapRegion(wal.log, int(WALMetaSize), mmap.RDWR, 0, 0)
	if nil != err {
		wal.log.Close()
		return nil, err
	}
	wal.readMeta()
	if wal.empty() {
		err = wal.log.Truncate(WALMetaSize)
		if nil != err {
			wal.log.Close()
			return wal, err
		}
	}
	wal.log.Seek(0, os.SEEK_END)
	return wal, nil
}

func newDiscardWAL() *WAL {
	wal := new(WAL)
	wal.discard = true
	return wal
}
