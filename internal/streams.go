package internal

import (
	"fmt"
	"sync"
)

const ControlStreamID = 0

type Streams struct {
	streams map[uint32]*Stream
	m       sync.Mutex
	conn    *Conn
}

func NewStreams(conn *Conn) *Streams {
	return &Streams{
		streams: make(map[uint32]*Stream),
		conn:    conn,
	}
}

// Create 함수는 주어진 스트림 ID로 새로운 Stream을 생성합니다. (스트림을 생성하고 메모리에서 관리)
func (ss *Streams) Create(streamID uint32) (*Stream, error) {
	ss.m.Lock()
	defer ss.m.Unlock()

	_, ok := ss.streams[streamID]
	if ok {
		return nil, fmt.Errorf("Stream already exists: StreamID = %d", streamID)
	}

	ss.streams[streamID] = newStream(streamID, ss.conn)

	return ss.streams[streamID], nil
}

func (ss *Streams) At(streamID uint32) (*Stream, error) {
	stream, ok := ss.streams[streamID]
	if !ok {
		return nil, fmt.Errorf("Stream not found: StreamID = %d", streamID)
	}

	return stream, nil
}
