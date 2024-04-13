package internal

import (
	"example/hello/message"
)

//var StreamMap = make(map[uint32]net.Conn)
//var mapMutex = &sync.Mutex{} // StreamMap을 안전하게 접근하기 위한 뮤텍스
//
//// CreateStream 함수는 주어진 스트림 ID와 연결을 StreamMap에 추가합니다.
//func CreateStream(streamID uint32, conn net.Conn) error {
//	mapMutex.Lock()
//	defer mapMutex.Unlock()
//
//	if _, exists := StreamMap[streamID]; exists {
//		// 이미 존재하는 스트림 ID인 경우 에러 반환
//		return errors.New("stream already exists")
//	}
//
//	// 스트림 맵에 스트림 ID와 연결 추가
//	StreamMap[streamID] = conn
//	return nil
//}

type Stream struct {
	streamID  uint32
	encTy     message.EncodingType
	handler   *streamHandler
	cmsg      ChunkMessage
	conn      *Conn
	chunkSize uint32
}

func newStream(streamID uint32, conn *Conn) *Stream {
	s := &Stream{
		streamID: streamID,
		encTy:    message.EncodingTypeAMF0, // Default AMF encoding type
		cmsg: ChunkMessage{
			StreamID: streamID,
		},
		conn: conn,
	}
	s.handler = newStreamHandler(s)
	return s
}

func (s *Stream) SetChunkSize(chunkSize uint32) error {
	s.chunkSize = chunkSize
	return nil
}

//func (s *Stream) Write(chunkStreamID int, timestamp uint32, msg message.Message) error {
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // TODO: Fix 5s
//	defer cancel()
//
//	s.cmsg.Message = msg
//	return s.streamer().Write(ctx, chunkStreamID, timestamp, &s.cmsg)
//}

//func (s *Stream) handle(chunkStreamID int, timestamp uint32, msg message.Message) error {
//	switch msg := msg.(type) {
//	case *message.DataMessage:
//		return s.handleData(chunkStreamID, timestamp, msg)
//
//	}
//
//	return nil
//}

//func (s *Stream) handleData(chunkStreamID int, timestamp uint32, dataMsg *message.DataMessage) error {
//	bodyDecoder := message.GetDataBodyDecoder(dataMsg.Name)
//	var value message.AMFConvertible
//	dataFrame, err := bodyDecoder(dataMsg.Body)
//	if err != nil {
//		return err
//	}
//
//	err := h.handler.onData(chunkStreamID, timestamp, dataMsg, dataFrame)
//	if err == internal.ErrPassThroughMsg {
//		return h.stream.userHandler().OnUnknownDataMessage(timestamp, dataMsg)
//	}
//	return err
//}
