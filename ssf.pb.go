// Code generated by protoc-gen-gogo.
// source: ssf.proto
// DO NOT EDIT!

/*
	Package ssf is a generated protocol buffer package.

	It is generated from these files:
		ssf.proto

	It has these top-level messages:
		EventHeader
		HeartBeat
		EventACK
*/
package ssf

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type EventType int32

const (
	EventType_EVENT_RAW       EventType = 1
	EventType_EVENT_HEARTBEAT EventType = 2
	EventType_EVENT_ACK       EventType = 3
)

var EventType_name = map[int32]string{
	1: "EVENT_RAW",
	2: "EVENT_HEARTBEAT",
	3: "EVENT_ACK",
}
var EventType_value = map[string]int32{
	"EVENT_RAW":       1,
	"EVENT_HEARTBEAT": 2,
	"EVENT_ACK":       3,
}

func (x EventType) Enum() *EventType {
	p := new(EventType)
	*p = x
	return p
}
func (x EventType) String() string {
	return proto.EnumName(EventType_name, int32(x))
}
func (x *EventType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(EventType_value, data, "EventType")
	if err != nil {
		return err
	}
	*x = EventType(value)
	return nil
}

type EventHeader struct {
	SequenceId       *uint64 `protobuf:"varint,1,opt,name=sequenceId" json:"sequenceId,omitempty"`
	HashCode         *uint64 `protobuf:"varint,2,opt,name=hashCode" json:"hashCode,omitempty"`
	NodeId           *int32  `protobuf:"varint,3,opt,name=nodeId" json:"nodeId,omitempty"`
	MsgType          *int32  `protobuf:"varint,4,opt,name=msgType" json:"msgType,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *EventHeader) Reset()         { *m = EventHeader{} }
func (m *EventHeader) String() string { return proto.CompactTextString(m) }
func (*EventHeader) ProtoMessage()    {}

func (m *EventHeader) GetSequenceId() uint64 {
	if m != nil && m.SequenceId != nil {
		return *m.SequenceId
	}
	return 0
}

func (m *EventHeader) GetHashCode() uint64 {
	if m != nil && m.HashCode != nil {
		return *m.HashCode
	}
	return 0
}

func (m *EventHeader) GetNodeId() int32 {
	if m != nil && m.NodeId != nil {
		return *m.NodeId
	}
	return 0
}

func (m *EventHeader) GetMsgType() int32 {
	if m != nil && m.MsgType != nil {
		return *m.MsgType
	}
	return 0
}

type HeartBeat struct {
	Req              *bool  `protobuf:"varint,1,opt,name=req" json:"req,omitempty"`
	Res              *bool  `protobuf:"varint,2,opt,name=res" json:"res,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *HeartBeat) Reset()         { *m = HeartBeat{} }
func (m *HeartBeat) String() string { return proto.CompactTextString(m) }
func (*HeartBeat) ProtoMessage()    {}

func (m *HeartBeat) GetReq() bool {
	if m != nil && m.Req != nil {
		return *m.Req
	}
	return false
}

func (m *HeartBeat) GetRes() bool {
	if m != nil && m.Res != nil {
		return *m.Res
	}
	return false
}

type EventACK struct {
	SequeceId        *int64  `protobuf:"varint,1,opt,name=sequeceId" json:"sequeceId,omitempty"`
	Mask             *uint64 `protobuf:"varint,2,opt,name=mask" json:"mask,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *EventACK) Reset()         { *m = EventACK{} }
func (m *EventACK) String() string { return proto.CompactTextString(m) }
func (*EventACK) ProtoMessage()    {}

func (m *EventACK) GetSequeceId() int64 {
	if m != nil && m.SequeceId != nil {
		return *m.SequeceId
	}
	return 0
}

func (m *EventACK) GetMask() uint64 {
	if m != nil && m.Mask != nil {
		return *m.Mask
	}
	return 0
}

func init() {
	proto.RegisterEnum("ssf.EventType", EventType_name, EventType_value)
}
func (m *EventHeader) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *EventHeader) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.SequenceId != nil {
		data[i] = 0x8
		i++
		i = encodeVarintSsf(data, i, uint64(*m.SequenceId))
	}
	if m.HashCode != nil {
		data[i] = 0x10
		i++
		i = encodeVarintSsf(data, i, uint64(*m.HashCode))
	}
	if m.NodeId != nil {
		data[i] = 0x18
		i++
		i = encodeVarintSsf(data, i, uint64(*m.NodeId))
	}
	if m.MsgType != nil {
		data[i] = 0x20
		i++
		i = encodeVarintSsf(data, i, uint64(*m.MsgType))
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *HeartBeat) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *HeartBeat) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Req != nil {
		data[i] = 0x8
		i++
		if *m.Req {
			data[i] = 1
		} else {
			data[i] = 0
		}
		i++
	}
	if m.Res != nil {
		data[i] = 0x10
		i++
		if *m.Res {
			data[i] = 1
		} else {
			data[i] = 0
		}
		i++
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *EventACK) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *EventACK) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.SequeceId != nil {
		data[i] = 0x8
		i++
		i = encodeVarintSsf(data, i, uint64(*m.SequeceId))
	}
	if m.Mask != nil {
		data[i] = 0x10
		i++
		i = encodeVarintSsf(data, i, uint64(*m.Mask))
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func encodeFixed64Ssf(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Ssf(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintSsf(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *EventHeader) Size() (n int) {
	var l int
	_ = l
	if m.SequenceId != nil {
		n += 1 + sovSsf(uint64(*m.SequenceId))
	}
	if m.HashCode != nil {
		n += 1 + sovSsf(uint64(*m.HashCode))
	}
	if m.NodeId != nil {
		n += 1 + sovSsf(uint64(*m.NodeId))
	}
	if m.MsgType != nil {
		n += 1 + sovSsf(uint64(*m.MsgType))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *HeartBeat) Size() (n int) {
	var l int
	_ = l
	if m.Req != nil {
		n += 2
	}
	if m.Res != nil {
		n += 2
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *EventACK) Size() (n int) {
	var l int
	_ = l
	if m.SequeceId != nil {
		n += 1 + sovSsf(uint64(*m.SequeceId))
	}
	if m.Mask != nil {
		n += 1 + sovSsf(uint64(*m.Mask))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovSsf(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozSsf(x uint64) (n int) {
	return sovSsf(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *EventHeader) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSsf
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventHeader: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventHeader: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SequenceId", wireType)
			}
			var v uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.SequenceId = &v
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field HashCode", wireType)
			}
			var v uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.HashCode = &v
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeId", wireType)
			}
			var v int32
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.NodeId = &v
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MsgType", wireType)
			}
			var v int32
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.MsgType = &v
		default:
			iNdEx = preIndex
			skippy, err := skipSsf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSsf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *HeartBeat) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSsf
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: HeartBeat: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: HeartBeat: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Req", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			b := bool(v != 0)
			m.Req = &b
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Res", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			b := bool(v != 0)
			m.Res = &b
		default:
			iNdEx = preIndex
			skippy, err := skipSsf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSsf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *EventACK) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSsf
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventACK: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventACK: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SequeceId", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.SequeceId = &v
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Mask", wireType)
			}
			var v uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Mask = &v
		default:
			iNdEx = preIndex
			skippy, err := skipSsf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSsf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSsf(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSsf
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSsf
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthSsf
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowSsf
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipSsf(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthSsf = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSsf   = fmt.Errorf("proto: integer overflow")
)