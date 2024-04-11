package internal

import (
	"example/hello/message"
	"fmt"
	"github.com/yutopp/go-amf0"
	"log"
)

type streamHandler struct {
	stream *Stream
}

func newStreamHandler(s *Stream) *streamHandler {
	return &streamHandler{
		stream: s,
	}
}

func (h *streamHandler) Handle(chunkStreamID int, timestamp uint32, msg message.Message) error {
	switch msg := msg.(type) {
	case *message.DataMessage:
		return h.handleData(chunkStreamID, timestamp, msg)

	case *message.CommandMessage:
		return h.handleCommand(chunkStreamID, timestamp, msg)

	case *message.SetChunkSize:
		return h.stream.SetChunkSize(msg.ChunkSize)

		//case *message.WinAckSize:
		//	l.Infof("Handle WinAckSize: Msg = %#v", msg)
		//	return h.stream.streamer().PeerState().SetAckWindowSize(msg.Size)
		//
		//default:
		//	err := h.handler.onMessage(chunkStreamID, timestamp, msg)
		//	if err == internal.ErrPassThroughMsg {
		//		return h.stream.userHandler().OnUnknownMessage(timestamp, msg)
		//	}
		//	return err
		//}
	}
	return nil
}

func (h *streamHandler) handleData(chunkStreamID int, timestamp uint32, dataMsg *message.DataMessage) error {
	//bodyDecoder := message.DataBodyDecoderFor(dataMsg.Name)
	//
	//amfDec := message.NewAMFDecoder(dataMsg.Body, dataMsg.Encoding)
	//var value message.AMFConvertible
	//if err := bodyDecoder(dataMsg.Body, amfDec, &value); err != nil {
	//	return err
	//}
	//
	//err := h.handler.onData(chunkStreamID, timestamp, dataMsg, value)
	//if err == internal.ErrPassThroughMsg {
	//	return h.stream.userHandler().OnUnknownDataMessage(timestamp, dataMsg)
	//}
	return nil
}

func (h *streamHandler) handleCommand(chunkStreamID int, timestamp uint32, cmdMsg *message.CommandMessage) error {
	amfDec := amf0.NewDecoder(cmdMsg.Body)

	var value message.AMFConvertible
	switch cmdMsg.CommandName {
	case "connect":
		if err := h.connect(amfDec, &value); err != nil {
			return err
		}
		if err := h.sendConnectSuccessResponse(); err != nil {
			return err
		}
		break
	case "releaseStream":
		if err := h.releaseStream(amfDec, &value); err != nil {
			return err
		}
		if

	}

	//err := h.handler.onCommand(chunkStreamID, timestamp, cmdMsg, value)
	//if err == internal.ErrPassThroughMsg {
	//	return h.stream.userHandler().OnUnknownCommandMessage(timestamp, cmdMsg)
	//}
	return nil
}

func (h *streamHandler) sendConnectSuccessResponse() error {
	successMessage := map[string]interface{}{
		"level":       "status",
		"code":        "NetConnection.Connect.Success",
		"description": "Connection succeeded.",
		"details": map[string]interface{}{
			"fmsVer":       "GO-RTMP/0,0,0,0",
			"capabilities": 31,
		},
		// AMF0에서는 ECMAArray 대신 map을 사용할 수 있습니다.
		"information": map[string]interface{}{
			"type":    "go-rtmp",
			"version": "master",
		},
	}

	// 로그를 통해 응답 내용 출력
	log.Printf("Connect: ResponseBody = %+v", successMessage)
	encoder := amf0.NewEncoder(h.stream.conn.rwc)

	if err := encoder.Encode(successMessage); err != nil {
		log.Printf("Failed to encode and send connect success response: %v", err)
		return err
	}

	//defer h.stream.conn.rwc.Close()
	fmt.Println("Connect success response sent successfully")
	return nil
}

type NetConnectionConnectResult struct {
	Properties  map[string]interface{}
	Information NetConnectionConnectResultInformation
}

// NetConnectionConnectResultInformation provides detailed information about the connect result
type NetConnectionConnectResultInformation struct {
	Level       string
	Code        string
	Description string
	Data        map[string]interface{}
}

func (h *streamHandler) connect(d *amf0.Decoder, value *message.AMFConvertible) error {
	var object map[string]interface{}
	if err := d.Decode(&object); err != nil {
		return fmt.Errorf("failed to decode 'connect': %v", err)
	}

	var cmd NetConnectionConnect
	if err := cmd.FromArgs(object); err != nil {
		return fmt.Errorf("failed to reconstruct 'connect : %v", err)
	}

	*value = &cmd
	return nil
}

func (h *streamHandler) releaseStream(d *amf0.Decoder, value *message.AMFConvertible) error {
	var commandObject interface{} // maybe nil
	if err := d.Decode(&commandObject); err != nil {
		return fmt.Errorf("Failed to decode 'releaseStream' args[0]")
	}
	var streamName string
	if err := d.Decode(&streamName); err != nil {
		return fmt.Errorf("Failed to decode 'releaseStream' args[1]")
	}

	var cmd NetConnectionReleaseStream
	if err := cmd.FromArgs(commandObject, streamName); err != nil {
		return fmt.Errorf("Failed to reconstruct 'releaseStream'")
	}

	*value = &cmd
	return nil
}

func (h *streamHandler) sendReleaseSuccessResponse() error {
	successMessage := map[string]interface{}{
		"level":       "status",
		"code":        "NetConnection.Connect.Success",
		"description": "Connection succeeded.",
		"details": map[string]interface{}{
			"fmsVer":       "GO-RTMP/0,0,0,0",
			"capabilities": 31,
		},
		// AMF0에서는 ECMAArray 대신 map을 사용할 수 있습니다.
		"information": map[string]interface{}{
			"type":    "go-rtmp",
			"version": "master",
		},
	}

	// 로그를 통해 응답 내용 출력
	log.Printf("Connect: ResponseBody = %+v", successMessage)
	encoder := amf0.NewEncoder(h.stream.conn.rwc)

	if err := encoder.Encode(successMessage); err != nil {
		log.Printf("Failed to encode and send connect success response: %v", err)
		return err
	}

	//defer h.stream.conn.rwc.Close()
	fmt.Println("Connect success response sent successfully")
	return nil
}