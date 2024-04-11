package internal

import "bytes"

type chunkBasicHeader struct {
	fmt           byte
	chunkStreamID int /* [0, 65599] */
}

type chunkMessageHeader struct {
	timestamp       uint32 // fmt = 0
	timestampDelta  uint32 // fmt = 1 | 2
	messageLength   uint32 // fmt = 0 | 1
	messageTypeID   byte   // fmt = 0 | 1
	messageStreamID uint32 // fmt = 0
}

type ChunkStreamReader struct {
	basicHeader   chunkBasicHeader
	messageHeader chunkMessageHeader

	timestamp       uint32
	timestampDelta  uint32
	messageLength   uint32 // max, 24bits
	messageTypeID   byte
	messageStreamID uint32

	buf       bytes.Buffer
	completed bool
}

func (r *ChunkStreamReader) Read(b []byte) (int, error) {
	return r.buf.Read(b)
}
