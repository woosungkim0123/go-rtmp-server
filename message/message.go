package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/yutopp/go-amf0"
	"io"
	"log"
)

type TypeID byte

const (
	TypeIDSetChunkSize            TypeID = 1
	TypeIDAbortMessage            TypeID = 2
	TypeIDAck                     TypeID = 3
	TypeIDUserCtrl                TypeID = 4
	TypeIDWinAckSize              TypeID = 5
	TypeIDSetPeerBandwidth        TypeID = 6
	TypeIDAudioMessage            TypeID = 8
	TypeIDVideoMessage            TypeID = 9
	TypeIDDataMessageAMF3         TypeID = 15
	TypeIDSharedObjectMessageAMF3 TypeID = 16
	TypeIDCommandMessageAMF3      TypeID = 17
	TypeIDDataMessageAMF0         TypeID = 18
	TypeIDSharedObjectMessageAMF0 TypeID = 19
	TypeIDCommandMessageAMF0      TypeID = 20
	TypeIDAggregateMessage        TypeID = 22
)

// Message
type Message interface {
	TypeID() TypeID
}

type DataMessage struct {
	Name     string
	Encoding EncodingType
	Body     io.Reader
}

func (m *DataMessage) TypeID() TypeID {
	switch m.Encoding {
	case EncodingTypeAMF3:
		return TypeIDDataMessageAMF3
	case EncodingTypeAMF0:
		return TypeIDDataMessageAMF0
	default:
		panic("Unreachable")
	}
}

// SetChunkSize (1)
type SetChunkSize struct {
	ChunkSize uint32
}

func (m *SetChunkSize) TypeID() TypeID {
	return TypeIDSetChunkSize
}

//var DataBodyDecoders = map[string]BodyDecoderFunc{
//	"@setDataFrame": DecodeBodyAtSetDataFrame,
//}

type CommandMessage struct {
	CommandName   string
	TransactionID int64
	Encoding      EncodingType
	Body          io.Reader
}

func (m *CommandMessage) TypeID() TypeID {
	switch m.Encoding {
	case EncodingTypeAMF3:
		return TypeIDCommandMessageAMF3
	case EncodingTypeAMF0:
		return TypeIDCommandMessageAMF0
	default:
		panic("Unreachable")
	}
}

type NetStreamSetDataFrame struct {
	Payload []byte
}

func (t *NetStreamSetDataFrame) FromArgs(args ...interface{}) {
	t.Payload = args[0].([]byte) // 배열
}

type BodyDecoderFunc func(r io.Reader) (*NetStreamSetDataFrame, error)

func DecodeBodyAtSetDataFrame(r io.Reader) (*NetStreamSetDataFrame, error) {
	// 버퍼에 읽은 데이터 복사
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("failed to decode '@setDataFrame' args[0]: %w", err)
	}

	// 이제 buf에 저장된 데이터를 NetStreamSetDataFrame의 Payload에 할당합니다.
	return &NetStreamSetDataFrame{Payload: buf.Bytes()}, nil
}

func GetDataBodyDecoder(name string) BodyDecoderFunc {
	if name == "@setDataFrame" {
		return DecodeBodyAtSetDataFrame
	}
	log.Printf("DataBodyDecoderFor: %s\n", name)
	return nil
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

func (dec *Decoder) Reset(r io.Reader) {
	dec.r = r
}

func (dec *Decoder) Decode(typeID TypeID, msg *Message) error {
	switch typeID {
	case TypeIDSetChunkSize:
		return dec.decodeSetChunkSize(msg)
	case TypeIDCommandMessageAMF0: // 20
		return dec.decodeCommandMessageAMF0(msg)
	default:
		return fmt.Errorf("Unexpected message type(decode): ID = %d", typeID)
	}
}

func (dec *Decoder) decodeSetChunkSize(msg *Message) error {
	buf := make([]byte, 4)
	if _, err := io.ReadAtLeast(dec.r, buf, 4); err != nil {
		return err
	}

	total := binary.BigEndian.Uint32(buf)

	bit := (total & 0x80000000) >> 31 // 0b1000,0000... >> 31
	chunkSize := total & 0x7fffffff   // 0b0111,1111...

	if bit != 0 {
		return fmt.Errorf("Invalid format: bit must be 0")
	}

	if chunkSize == 0 {
		return fmt.Errorf("Invalid format: chunk size is 0")
	}

	*msg = &SetChunkSize{
		ChunkSize: chunkSize,
	}

	return nil
}

func (dec *Decoder) decodeCommandMessageAMF0(msg *Message) error {
	if err := dec.decodeCommandMessage(msg, func(r io.Reader) (AMFDecoder, EncodingType) {
		return amf0.NewDecoder(r), EncodingTypeAMF0
	}); err != nil {
		return err
	}

	return nil
}

func (dec *Decoder) decodeCommandMessage(msg *Message, f func(r io.Reader) (AMFDecoder, EncodingType)) error {
	d, encTy := f(dec.r)

	var name string
	if err := d.Decode(&name); err != nil {
		return fmt.Errorf("Failed to decode commandName: %w", err)
	}

	var transactionID int64
	if err := d.Decode(&transactionID); err != nil {
		return fmt.Errorf("Failed to decode transactionID: %w", err)
	}

	*msg = &CommandMessage{
		CommandName:   name,
		TransactionID: transactionID,
		Encoding:      encTy,
		Body:          dec.r, // Share an ownership of the reader
	}

	return nil
}
