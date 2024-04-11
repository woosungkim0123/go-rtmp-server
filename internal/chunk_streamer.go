package internal

import (
	"encoding/binary"
	"example/hello/message"
	"io"
	"log"
	"sync"
)

type ChunkMessage struct {
	StreamID uint32
	Message  message.Message
}

type ChunkStreamer struct {
	r *ChunkStreamerReader

	messageDecoder *message.Decoder

	readers map[int]*ChunkStreamReader
	mu      sync.Mutex

	err  error
	done chan struct{}

	controlStreamWriter func(chunkStreamID int, timestamp uint32, msg message.Message) error

	cacheBuffer []byte
}

func NewChunkStreamer(r io.Reader, w io.Writer) *ChunkStreamer {

	cs := &ChunkStreamer{
		r: &ChunkStreamerReader{
			reader: r,
		},

		readers: make(map[int]*ChunkStreamReader),

		messageDecoder: message.NewDecoder(nil),

		done: make(chan struct{}),

		cacheBuffer: make([]byte, 64*1024), // cache 64KB
	}
	// TODO 비동기 처리
	return cs
}

func (cs *ChunkStreamer) Read(cmsg *ChunkMessage) (int, uint32, error) {
	reader, err := cs.NewChunkReader()
	if err != nil {
		return 0, 0, err
	}

	cs.messageDecoder.Reset(reader)
	if err := cs.messageDecoder.Decode(message.TypeID(reader.messageTypeID), &cmsg.Message); err != nil {
		return 0, 0, err
	}

	cmsg.StreamID = reader.messageStreamID

	return reader.basicHeader.chunkStreamID, uint32(reader.timestamp), nil
}

func (cs *ChunkStreamer) NewChunkReader() (*ChunkStreamReader, error) {
again:
	reader, err := cs.readChunk()
	if err != nil {
		return nil, err
	}

	if !reader.completed {
		goto again
	}
	return reader, nil
}

func (cs *ChunkStreamer) readChunk() (*ChunkStreamReader, error) {
	var bh chunkBasicHeader
	if err := decodeChunkBasicHeader(cs.r, cs.cacheBuffer, &bh); err != nil {
		return nil, err
	}
	//cs.logger.Debugf("(READ) BasicHeader = %+v", bh)

	var mh chunkMessageHeader
	if err := decodeChunkMessageHeader(cs.r, bh.fmt, cs.cacheBuffer, &mh); err != nil {
		return nil, err
	}
	//cs.logger.Debugf("(READ) MessageHeader = %+v", mh)

	reader, err := cs.prepareChunkReader(bh.chunkStreamID)
	if err != nil {
		return nil, err
	}

	if reader.completed {
		reader.buf.Reset()
		reader.completed = false
	}

	reader.basicHeader = bh
	reader.messageHeader = mh

	switch bh.fmt {
	case 0:
		reader.timestamp = mh.timestamp
		reader.timestampDelta = 0 // reset
		reader.messageLength = mh.messageLength
		reader.messageTypeID = mh.messageTypeID
		reader.messageStreamID = mh.messageStreamID
	case 1:
		reader.timestampDelta = mh.timestampDelta
		reader.messageLength = mh.messageLength
		reader.messageTypeID = mh.messageTypeID
	case 2:
		reader.timestampDelta = mh.timestampDelta

	case 3:
	// DO NOTHING

	default:
		panic("unsupported chunk") // TODO: fix
	}

	log.Printf("(READ) MessageLength = %d, Current = %d", reader.messageLength, reader.buf.Len())

	expectLen := int(reader.messageLength) - reader.buf.Len()
	if expectLen <= 0 {
		panic("invalid state") // TODO fix
	}

	lr := io.LimitReader(cs.r, int64(expectLen))
	if _, err := io.CopyBuffer(&reader.buf, lr, cs.cacheBuffer); err != nil {
		return nil, err
	}
	//cs.logger.Debugf("(READ) Buffer: %+v", reader.buf.Bytes())

	if int(reader.messageLength)-reader.buf.Len() != 0 {
		// fragmented
		return reader, nil
	}

	// read completed, update timestamp
	reader.timestamp += reader.timestampDelta
	reader.completed = true

	return reader, nil
}

func decodeChunkBasicHeader(r io.Reader, buf []byte, bh *chunkBasicHeader) error {
	if buf == nil || len(buf) < 3 {
		buf = make([]byte, 3)
	}

	if _, err := io.ReadAtLeast(r, buf[:1], 1); err != nil {
		return err
	}

	fmtTy := (buf[0] & 0xc0) >> 6 // 0b11000000 >> 6
	csID := int(buf[0] & 0x3f)    // 0b00111111

	switch csID {
	case 0:
		if _, err := io.ReadAtLeast(r, buf[1:2], 1); err != nil {
			return err
		}
		csID = int(buf[1]) + 64

	case 1:
		if _, err := io.ReadAtLeast(r, buf[1:], 2); err != nil {
			return err
		}
		csID = int(buf[2])*256 + int(buf[1]) + 64
	}

	bh.fmt = fmtTy
	bh.chunkStreamID = csID

	return nil
}

func decodeChunkMessageHeader(r io.Reader, fmt byte, buf []byte, mh *chunkMessageHeader) error {
	if buf == nil || len(buf) < 11 {
		buf = make([]byte, 11)
	}
	cache32bits := make([]byte, 4)

	switch fmt {
	case 0:
		if _, err := io.ReadAtLeast(r, buf[:11], 11); err != nil {
			return err
		}

		copy(cache32bits[1:], buf[0:3]) // 24bits BE
		mh.timestamp = binary.BigEndian.Uint32(cache32bits)
		copy(cache32bits[1:], buf[3:6]) // 24bits BE
		mh.messageLength = binary.BigEndian.Uint32(cache32bits)
		mh.messageTypeID = buf[6]                                  // 8bits
		mh.messageStreamID = binary.LittleEndian.Uint32(buf[7:11]) // 32bits

		if mh.timestamp == 0xffffff {
			_, err := io.ReadAtLeast(r, cache32bits, 4)
			if err != nil {
				return err
			}
			mh.timestamp = binary.BigEndian.Uint32(cache32bits)
		}

	case 1:
		if _, err := io.ReadAtLeast(r, buf[:7], 7); err != nil {
			return err
		}

		copy(cache32bits[1:], buf[0:3]) // 24bits BE
		mh.timestampDelta = binary.BigEndian.Uint32(cache32bits)
		copy(cache32bits[1:], buf[3:6]) // 24bits BE
		mh.messageLength = binary.BigEndian.Uint32(cache32bits)
		mh.messageTypeID = buf[6] // 8bits

		if mh.timestampDelta == 0xffffff {
			_, err := io.ReadAtLeast(r, cache32bits, 4)
			if err != nil {
				return err
			}
			mh.timestampDelta = binary.BigEndian.Uint32(cache32bits)
		}

	case 2:
		if _, err := io.ReadAtLeast(r, buf[:3], 3); err != nil {
			return err
		}

		copy(cache32bits[1:], buf[0:3]) // 24bits BE
		mh.timestampDelta = binary.BigEndian.Uint32(cache32bits)

		if mh.timestampDelta == 0xffffff {
			_, err := io.ReadAtLeast(r, cache32bits, 4)
			if err != nil {
				return err
			}
			mh.timestampDelta = binary.BigEndian.Uint32(cache32bits)
		}

	case 3:
		// DO NOTHING

	default:
		panic("Unexpected fmt")
	}

	return nil
}

func (cs *ChunkStreamer) prepareChunkReader(chunkStreamID int) (*ChunkStreamReader, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reader, ok := cs.readers[chunkStreamID]
	if !ok {
		reader = &ChunkStreamReader{}
		cs.readers[chunkStreamID] = reader
	}

	return reader, nil
}
