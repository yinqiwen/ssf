package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/edsrzf/mmap-go"
	"github.com/golang/glog"
	"io"
	"os"
	"sync"
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
	wal.replayLock.Lock()
	defer wal.replayLock.Unlock()

	wal.logLock.Lock()
	readedOffset := wal.meta.readedOffset
	fileSize := wal.meta.fileSize
	wal.logLock.Unlock()

	if readedOffset < fileSize {
		f, err := os.Open(wal.logPath)
		if nil != err {
			return
		}
		f.Seek(WALMetaSize+wal.meta.readedOffset, 0)
		replayBuf := make([]byte, 0)
		b := make([]byte, 4096)
		for readedOffset < fileSize {
			n, err := f.Read(b)
			if 0 == n {
				break
			}
			if len(replayBuf) > 0 {
				replayBuf = append(replayBuf, b[0:n]...)
			} else {
				replayBuf = b[0:n]
			}
			cursor := consumeEvents(replayBuf)
			if cursor > 0 {
				wbuf := make([]byte, cursor)
				copy(wbuf, replayBuf[0:cursor])
				replayBuf = replayBuf[cursor:]
				_, err = writer.Write(wbuf)
				if nil != err {
					break
				}
				wal.meta.readedOffset += int64(cursor)
				readedOffset += int64(cursor)
			}

		}
		f.Close()
		wal.logLock.Lock()
		if wal.meta.readedOffset == wal.meta.fileSize {
			wal.meta.readedOffset = 0
			wal.meta.fileSize = 0
			wal.log.Truncate(WALMetaSize)
			wal.log.Seek(0, os.SEEK_END)
			glog.Infof("Clear wal for virtual node:%d", wal.nodeId)
		}
		wal.syncMeta()
		wal.logLock.Unlock()
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
		wal.log.Truncate(WALMetaSize)
	}
	wal.log.Seek(0, os.SEEK_SET)
	wal.mapedMeta, err = mmap.MapRegion(wal.log, int(WALMetaSize), mmap.RDWR, 0, 0)
	if nil != err {
		wal.log.Close()
		return nil, err
	}
	wal.readMeta()
	wal.log.Seek(0, os.SEEK_END)
	return wal, nil
}

func newDiscardWAL() *WAL {
	wal := new(WAL)
	wal.discard = true
	return wal
}