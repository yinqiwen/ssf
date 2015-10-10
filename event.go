package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"io"
	"reflect"
	"sync"
)

type Event struct {
	Sequence uint64
	HashCode uint64
	MsgType  int32
	Msg      proto.Message
}

var int2Reflect map[int]reflect.Type = make(map[int]reflect.Type)
var reflect2Int map[reflect.Type]int = make(map[reflect.Type]int)
var lk sync.Mutex // guards maps

func RegisterEvent(eventType int, ev proto.Message) error {
	t := reflect.TypeOf(ev).Elem()
	lk.Lock()
	defer lk.Unlock()
	if _, ok := int2Reflect[eventType]; ok {
		return fmt.Errorf("Duplicate registration with type:%d", eventType)
	}
	if oldInt, ok := reflect2Int[t]; ok && eventType != oldInt {
		return fmt.Errorf("Same type:%T already registed as:%d", ev, eventType)
	}
	int2Reflect[eventType] = t
	reflect2Int[t] = eventType
	return nil
}

func getEventByType(eventType int) proto.Message {
	lk.Lock()
	defer lk.Unlock()
	oldType, ok := int2Reflect[eventType]
	if !ok {
		return nil
	}
	return reflect.New(oldType).Interface().(proto.Message)
}

func getEventType(ev proto.Message) int {
	lk.Lock()
	defer lk.Unlock()
	oldType, ok := reflect2Int[reflect.TypeOf(ev).Elem()]
	if !ok {
		return -1
	}
	return oldType
}

func readEvent(reader io.Reader) (*Event, error) {
	var headerLen uint32
	err := binary.Read(reader, binary.LittleEndian, &headerLen)
	if nil != err {
		return nil, err
	}
	var header EventHeader
	headerBuf := make([]byte, headerLen)
	n, err := reader.Read(headerBuf)
	if nil != err || n != len(headerBuf) {
		return nil, err
	}
	err = header.Unmarshal(headerBuf)
	if nil != err {
		return nil, err
	}
	dataBuf := make([]byte, header.GetMsgLen())
	msg := getEventByType(int(header.GetMsgType()))
	if nil == msg {
		return nil, fmt.Errorf("No msg type found for:%d", header.GetMsgType())
	}
	err = proto.Unmarshal(dataBuf, msg)
	if nil != err {
		return nil, err
	}
	ev := new(Event)
	ev.Msg = msg
	ev.HashCode = header.GetHashCode()
	ev.MsgType = header.GetMsgType()
	ev.Sequence = header.GetSequenceId()
	return ev, nil
}

func writeEvent(ev *Event, writer io.Writer) error {
	var buf bytes.Buffer
	dataBuf, err := proto.Marshal(ev.Msg)
	if nil != err {
		return err
	}
	var header EventHeader
	headbuf, _ := proto.Marshal(&header)
	headerLen := uint32(len(headbuf))
	binary.Write(&buf, binary.LittleEndian, &headerLen)
	buf.Write(headbuf)
	buf.Write(dataBuf)
	_, err = buf.WriteTo(writer)
	return err
}
