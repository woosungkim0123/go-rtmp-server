package message

import "io"

type EncodingType uint8

const (
	EncodingTypeAMF0 EncodingType = 0
	EncodingTypeAMF3 EncodingType = 3
)

type AMFDecoder interface {
	Decode(interface{}) error
	Reset(r io.Reader)
}

type AMFConvertible interface {
	FromArgs(args ...interface{}) error
	ToArgs(ty EncodingType) ([]interface{}, error)
}
