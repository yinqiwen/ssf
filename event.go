package ssf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/protobuf/proto"
)

var MAGIC_EVENT_HEADER []byte = []byte("SSFE")
var MAGIC_OTSC_HEADER []byte = []byte("OTSC")

type Event struct {
	Length   uint32
	Sequence uint64
	HashCode uint64
	MsgType  string
	Msg      proto.Message
	Raw      []byte
}

func (ev *Event) Decode() error {
	if nil != ev.Msg {
		return nil
	}
	t := proto.MessageType(ev.MsgType)
	if nil == t {
		return fmt.Errorf("No msg type found for:%s", ev.MsgType)
	}
	msg := reflect.New(t.Elem()).Interface().(proto.Message)
	pos := (ev.Length & 0xFF) + 8
	err := proto.Unmarshal(ev.Raw[pos:], msg)
	if nil != err {
		return err
	}
	ev.Msg = msg
	return nil
}

//EventHandler define ssf event handler interface
type EventHandler interface {
	OnEvent(event *Event) *Event
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
	var header EventHeader
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
	err = header.Unmarshal(allbuf[8 : headerLen+8])
	if nil != err {
		return nil, err
	}
	ev := new(Event)
	ev.Length = lengthHeader
	ev.HashCode = header.GetHashCode()
	ev.MsgType = header.GetMsgType()
	ev.Sequence = header.GetSequenceId()
	ev.Raw = allbuf
	return ev, nil
}

func readEvent(reader io.Reader, ignoreMagic, decodeBody bool) (*Event, error) {
	ev, err := readRawEvent(reader, ignoreMagic)
	if nil != err {
		return nil, err
	}
	if decodeBody {
		err = ev.Decode()
		if nil != err {
			return nil, err
		}
	}
	return ev, nil
}

//WriteEvent encode&write message to writer
func WriteEvent(msg proto.Message, hashCode, seq uint64, writer io.Writer) error {
	var event Event
	event.HashCode = hashCode
	event.Sequence = seq
	event.Msg = msg
	return writeEvent(&event, writer)
}

func writeEvent(ev *Event, writer io.Writer) error {
	if len(ev.Raw) > 0 {
		_, err := writer.Write(ev.Raw)
		return err
	}
	var buf bytes.Buffer
	dataBuf, err := proto.Marshal(ev.Msg)
	if nil != err {
		return err
	}

	var header EventHeader
	ev.MsgType = proto.MessageName(ev.Msg)
	header.MsgType = &(ev.MsgType)
	header.HashCode = &ev.HashCode
	header.SequenceId = &ev.Sequence
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

func init() {
	proto.RegisterType((*RawMessage)(nil), "ssf.RawMessage")
}
