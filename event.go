package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/gogo/protobuf/proto"
)

var MAGIC_EVENT_HEADER []byte = []byte("SSFE")
var MAGIC_OTSC_HEADER []byte = []byte("OTSC")

type Event struct {
	Sequence uint64
	HashCode uint64
	MsgType  int32
	Msg      proto.Message
}

type RawMessage struct {
	raw []byte
}

func (m *RawMessage) Reset() {
}
func (m *RawMessage) String() string {
	return string(m.raw)
}
func (m *RawMessage) ProtoMessage() {
}
func (m *RawMessage) Marshal() ([]byte, error) {
	return m.raw, nil
}
func (m *RawMessage) Unmarshal(p []byte) error {
	m.raw = p
	return nil
}
func (m *RawMessage) Data() []byte {
	return m.raw
}

//NewRawMessage create a []byte proto message
func NewRawMessage(str string) *RawMessage {
	msg := new(RawMessage)
	msg.raw = []byte(str)
	return msg
}

var int2Reflect = make(map[int32]reflect.Type)
var reflect2Int = make(map[reflect.Type]int32)
var lk sync.Mutex // guards maps

//RegisterEvent register message with int type
func RegisterEvent(eventType int32, ev proto.Message) {
	t := reflect.TypeOf(ev).Elem()
	lk.Lock()
	defer lk.Unlock()
	if _, ok := int2Reflect[eventType]; ok {
		panic(fmt.Errorf("Duplicate registration with type:%d", eventType))
	}
	if oldInt, ok := reflect2Int[t]; ok && eventType != oldInt {
		panic(fmt.Errorf("Same type:%T already registed as:%d", ev, eventType))
	}
	int2Reflect[eventType] = t
	reflect2Int[t] = eventType
}

func GetEventByType(eventType int32) proto.Message {
	lk.Lock()
	defer lk.Unlock()
	oldType, ok := int2Reflect[eventType]
	if !ok {
		return nil
	}

	return reflect.New(oldType).Interface().(proto.Message)
}

//GetEventType return the registed int type of proto message
// return -1 if no such message registed
func GetEventType(ev proto.Message) int32 {
	lk.Lock()
	defer lk.Unlock()
	oldType, ok := reflect2Int[reflect.TypeOf(ev).Elem()]
	if !ok {
		return -1
	}
	return oldType
}

func readMagicHeader(reader io.Reader, buf []byte) ([]byte, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("Invalid buf space")
	}
	_, err := io.ReadFull(reader, buf)
	if nil != err {
		return nil, err
	}
	return buf[0:4], nil
}

func readEvent(reader io.Reader, ignoreMagic bool) (*Event, error) {
	var lenbuf []byte
	if !ignoreMagic {
		lenbuf = make([]byte, 8)
		magic, err := readMagicHeader(reader, lenbuf)
		if nil != err {
			return nil, err
		}
		if !bytes.Equal(magic, MAGIC_EVENT_HEADER) {
			return nil, fmt.Errorf("Invalid magic header:%s", string(magic))
		}
		lenbuf = lenbuf[4:8]
	} else {
		lenbuf = make([]byte, 4)
		_, err := io.ReadFull(reader, lenbuf)
		if nil != err {
			return nil, err
		}
	}

	var lengthHeader uint32
	binary.Read(bytes.NewReader(lenbuf), binary.LittleEndian, &lengthHeader)
	msgLen := lengthHeader >> 8
	headerLen := (lengthHeader & 0xFF)
	if headerLen > 100 || msgLen > 512*1024*1024 {
		return nil, fmt.Errorf("Too large msg header len:%d or body len:%d", headerLen, msgLen)
	}
	var err error
	var header EventHeader
	headerBuf := make([]byte, headerLen)
	_, err = io.ReadFull(reader, headerBuf)
	if nil != err {
		return nil, err
	}
	err = header.Unmarshal(headerBuf)
	if nil != err {
		return nil, err
	}
	dataBuf := make([]byte, msgLen)
	_, err = io.ReadFull(reader, dataBuf)
	if nil != err {
		return nil, err
	}
	msg := GetEventByType(header.GetMsgType())
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

//WriteEvent encode&write message to writer
func WriteEvent(msg proto.Message, hashCode uint64, writer io.Writer) error {
	var event Event
	event.HashCode = hashCode
	event.MsgType = int32(GetEventType(msg))
	event.Msg = msg
	return writeEvent(&event, writer)
}

func writeEvent(ev *Event, writer io.Writer) error {
	var buf bytes.Buffer
	dataBuf, err := proto.Marshal(ev.Msg)
	if nil != err {
		return err
	}

	var header EventHeader
	header.MsgType = &(ev.MsgType)
	header.HashCode = &ev.HashCode
	headbuf, _ := proto.Marshal(&header)
	headerLen := uint32(len(headbuf))

	// length = msglength(3byte) + headerlen(1byte)
	length := uint32(len(dataBuf))
	length = (length << 8) + headerLen
	buf.Write(MAGIC_EVENT_HEADER)
	binary.Write(&buf, binary.LittleEndian, length)

	buf.Write(headbuf)
	buf.Write(dataBuf)
	_, err = buf.WriteTo(writer)

	return err
}

// // return the index which have started is not a full event content
// func consumeEvents(p []byte) int {
// 	cursor := 0
// 	for {
// 		if len(p) <= 8 {
// 			return cursor
// 		}
// 		if !bytes.Equal(p[0:4], MAGIC_EVENT_HEADER) {
// 			return cursor
// 		}
// 		var lengthHeader uint32
// 		binary.Read(bytes.NewReader(p[4:8]), binary.LittleEndian, &lengthHeader)
// 		msgLen := lengthHeader >> 8
// 		headerLen := (lengthHeader & 0xFF)
// 		if uint32(len(p)-8) < (msgLen + headerLen) {
// 			return cursor
// 		}
// 		cursor += int(msgLen + headerLen + 8)
// 		p = p[(msgLen + headerLen + 8):]
// 	}
// 	return cursor
// }

func init() {
	raw := RawMessage{}
	hb := HeartBeat{}
	RegisterEvent(int32(EventType_EVENT_RAW), &raw)
	RegisterEvent(int32(EventType_EVENT_HEARTBEAT), &hb)
}
