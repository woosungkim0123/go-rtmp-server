package internal

type rtmpChunk struct {
	header   *chunkHeader
	clock    uint32
	delta    uint32
	payload  []byte
	bytes    int
	capacity uint32
}

type chunkHeader struct {
	fmt                  uint8
	csID                 uint32
	timestamp            uint32
	length               uint32
	messageType          uint8
	messageStreamID      uint32
	hasExtendedTimestamp bool
}

func (c *Connection) createRtmpChunk(fmt uint8, csID uint32) *rtmpChunk {
	header := &chunkHeader{
		fmt:                  fmt,
		csID:                 csID,
		timestamp:            0,
		length:               0,
		messageType:          0,
		messageStreamID:      0,
		hasExtendedTimestamp: false,
	}

	chunk := &rtmpChunk{
		header:   header,
		clock:    0,
		delta:    0,
		bytes:    0,
		capacity: 0,
	}

	return chunk
}
