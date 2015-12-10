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
	nodeId    int32
	meta      WALMeta
	logPath   string
	log       *os.File
	mapedMeta mmap.MMap
	logLock   sync.Mutex
	discard   bool
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

func (wal *WAL) sync() error {
	if wal.discard {
		return nil
	}
	fst, _ := wal.log.Stat()
	for wal.meta.fileSize+WALMetaSize > fst.Size() {
		glog.Warningf("Sync WAL[%d] file, since cached file size:%d is larger than stat size:%d", wal.meta.fileSize+WALMetaSize, fst.Size())
		wal.log.Sync()
		fst, _ = wal.log.Stat()
	}
	return nil
}

func (wal *WAL) cachedDataSize() int64 {
	return wal.meta.fileSize - wal.meta.readedOffset
}

func (wal *WAL) nextSequenceID() uint64 {
	if wal.discard {
		return 0
	}
	return 0
}
func (wal *WAL) replay(writer io.Writer) error {
	if wal.discard || nil == writer {
		return nil
	}
	b := make([]byte, 8192)
	glog.Infof("Start replay wal for virtual node:%d with data size [%d:%d]", wal.nodeId, wal.meta.readedOffset, wal.meta.fileSize)
	wal.logLock.Lock()
	defer wal.logLock.Unlock()
	wal.sync()
	for wal.meta.readedOffset < wal.meta.fileSize {
		n, err := wal.log.ReadAt(b, WALMetaSize+wal.meta.readedOffset)
		if n <= 0 {
			if nil != err {
				fst, _ := wal.log.Stat()
				glog.Warningf("Read wal log to node:%d failed:%v, the file stat size is %d, cached size is %d", wal.nodeId, err, fst.Size(), wal.meta.fileSize+WALMetaSize)
			}
			break
		}
		_, err = io.Copy(writer, bytes.NewBuffer(b[0:n]))
		if nil != err {
			glog.Warningf("Write replay log to node:%d failed:%v", wal.nodeId, err)
			return err
		}
		wal.meta.readedOffset += int64(n)
	}
	glog.Infof("Stop replay wal for virtual node:%d with current data size [%d:%d]", wal.nodeId, wal.meta.readedOffset, wal.meta.fileSize)
	if wal.meta.readedOffset == wal.meta.fileSize {
		err := wal.log.Truncate(WALMetaSize)
		if err == nil {
			wal.meta.readedOffset = 0
			wal.meta.fileSize = 0
			wal.log.Seek(0, os.SEEK_END)
			glog.Infof("WAL[%d] truncate to empty.", wal.nodeId)
		} else {
			glog.Errorf("Failed to truncate wal:%d for reason:%v", wal.nodeId, err)
		}
	}
	wal.syncMeta()
	return nil
}

func newWAL(nodeID int32) (*WAL, error) {
	wal := new(WAL)
	wal.nodeId = nodeID
	wal.logPath = fmt.Sprintf("%s/wal/node_%d.wal", ssfCfg.ProcHome, nodeID)
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
	fstat, _ = wal.log.Stat()
	if wal.meta.fileSize+WALMetaSize > fstat.Size() {
		wal.meta.fileSize = fstat.Size() - WALMetaSize
		wal.syncMeta()
	}
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
