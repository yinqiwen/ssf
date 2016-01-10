package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

var MAGIC_EVENT_HEADER []byte = []byte("SSFE")
var MAGIC_OTSC_HEADER []byte = []byte("OTSC")

type Event struct {
	EventHeader
	Msg proto.Message

	length uint32
	raw    []byte
}

func (ev *Event) decode() error {
	if nil != ev.Msg {
		return nil
	}
	t := proto.MessageType(ev.GetMsgType())
	if nil == t {
		return fmt.Errorf("No msg type found for:%s", ev.GetMsgType())
	}
	msg := reflect.New(t.Elem()).Interface().(proto.Message)
	pos := (ev.length & 0xFF) + 8
	err := proto.Unmarshal(ev.raw[pos:], msg)
	if nil != err {
		return err
	}
	ev.Msg = msg
	return nil
}

//handle event or IPC channel
type ipcEventHandler interface {
	OnEvent(event *Event, conn io.ReadWriteCloser)
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

func readRawEvent(reader io.Reader, ignoreMagic bool) (*Event, error) {
	allbuf := make([]byte, 256)
	lenbuf := allbuf[0:8]
	if !ignoreMagic {
		magic, err := readMagicHeader(reader, lenbuf)
		if nil != err {
			return nil, err
		}
		if !bytes.Equal(magic, MAGIC_EVENT_HEADER) {
			return nil, fmt.Errorf("Invalid magic header:%s", string(magic))
		}
	} else {
		copy(lenbuf[0:4], MAGIC_EVENT_HEADER)
		_, err := io.ReadFull(reader, lenbuf[4:8])
		if nil != err {
			return nil, err
		}
	}

	var lengthHeader uint32
	binary.Read(bytes.NewReader(lenbuf[4:8]), binary.LittleEndian, &lengthHeader)
	msgLen := lengthHeader >> 8
	headerLen := (lengthHeader & 0xFF)
	if headerLen > 128 || msgLen > 512*1024*1024 {
		return nil, fmt.Errorf("Too large msg header len:%d or body len:%d", headerLen, msgLen)
	}
	var err error
	ev := new(Event)
	//var header EventHeader
	totalLen := int(headerLen + msgLen + 8)
	if len(allbuf) < totalLen {
		allbuf = append(allbuf, make([]byte, totalLen-len(allbuf))...)
	} else {
		allbuf = allbuf[0:totalLen]
	}
	_, err = io.ReadFull(reader, allbuf[8:])
	if nil != err {
		return nil, err
	}
	err = ev.Unmarshal(allbuf[8 : headerLen+8])
	if nil != err {
		return nil, err
	}

	ev.length = lengthHeader
	ev.raw = allbuf
	return ev, nil
}

func readEvent(reader io.Reader, ignoreMagic, decodeBody bool) (*Event, error) {
	ev, err := readRawEvent(reader, ignoreMagic)
	if nil != err {
		return nil, err
	}
	if decodeBody {
		err = ev.decode()
		if nil != err {
			return nil, err
		}
	}
	return ev, nil
}

func notify(msg proto.Message, hashCode uint64, writer io.Writer) error {
	var event Event
	event.HashCode = &hashCode
	event.Msg = msg
	event.Type = EventType_NOTIFY.Enum()
	return writeEvent(&event, writer)
}

var clientSessions = make(map[uint64]chan proto.Message)
var clientSessionIDSeed = uint64(0)
var clientSessionsLock sync.Mutex

func addClientRPCSession() (uint64, chan proto.Message) {
	clientSessionsLock.Lock()
	defer clientSessionsLock.Unlock()
	id := clientSessionIDSeed
	clientSessionIDSeed++
	ch := make(chan proto.Message)
	clientSessions[id] = ch
	return id, ch
}

func triggerClientSessionRes(res proto.Message, hashCode uint64) {
	clientSessionsLock.Lock()
	defer clientSessionsLock.Unlock()
	if nil != res {
		ch, ok := clientSessions[hashCode]
		if !ok {
			return
		}
		ch <- res
	}
	delete(clientSessions, hashCode)
}

func rpc(processor string, request proto.Message, timeout time.Duration, wr io.Writer) (proto.Message, error) {
	id, rch := addClientRPCSession()
	defer close(rch)
	var event Event
	event.HashCode = &id
	event.Msg = request
	event.Type = EventType_REQUEST.Enum()
	proc := getProcessorName()
	event.From = &proc
	event.To = &processor
	err := writeEvent(&event, wr)
	if nil != err {
		return nil, err
	}
	timeoutTicker := time.NewTicker(timeout).C
	select {
	case res := <-rch:
		return res, nil
	case <-timeoutTicker:
		triggerClientSessionRes(nil, id)
		return nil, ErrCommandTimeout
	}
	return nil, ErrCommandTimeout
}

func response(res proto.Message, request *Event, wr io.Writer) error {
	var event Event
	event.HashCode = request.HashCode
	event.Msg = res
	event.Type = EventType_RESPONSE.Enum()
	event.From = request.To
	event.To = request.From
	return writeEvent(&event, wr)
}

// func rpc(msg proto.Message, writer io.Writer) error {
// 	var event Event
// 	//event.HashCode = &hashCode
// 	event.Msg = msg

// }

//WriteEvent encode&write message to writer
// func WriteEvent(msg proto.Message, hashCode, seq uint64, writer io.Writer) error {
// 	var event Event
// 	event.HashCode = &hashCode
// 	event.SequenceId = &seq
// 	event.Msg = msg
// 	return writeEvent(&event, writer)
// }

func writeEvent(ev *Event, writer io.Writer) error {
	if len(ev.raw) > 0 {
		_, err := writer.Write(ev.raw)
		return err
	}
	var buf bytes.Buffer
	dataBuf, err := proto.Marshal(ev.Msg)
	if nil != err {
		return err
	}
	msgtype := proto.MessageName(ev.Msg)
	ev.MsgType = &msgtype
	headbuf, _ := proto.Marshal(ev)
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

func init() {
	proto.RegisterType((*RawMessage)(nil), "ssf.RawMessage")
}
