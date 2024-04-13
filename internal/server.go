package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

type Message struct {
	Type    string // 메시지 타입
	Payload []byte // 메시지 내용
}

func Serve(c *Conn) error {

	if err := Handshake1(c); err != nil {
		log.Printf("Failed to handshake: %+v", err)
		panic(err)
	}

	_, err := c.streams.Create(uint32(ControlStreamID))

	if err != nil {
		log.Printf("Failed to create control stream: %+v", err)
		panic(err)
	}

	return c.runHandleMessageLoop()
}

//// AMF0 메시지를 읽는 곳
//var commandName string
//var transactionID float64
//var obj interface{}
//
//commandName, err := amf.ReadString(conn.bReader)
//if err != nil {
//	log.Printf("Failed to read command name: %+v", err)
//	return err
//}
//transactionID, err2 := amf.ReadDouble(conn.bReader)
//if err2 != nil {
//	log.Printf("Failed to read transaction ID: %+v", err)
//	return err
//}
//
//// 연결 매개변수 객체 읽기
//obj, err3 := amf.ReadValue(conn.bReader)
//if err3 != nil {
//	log.Printf("Failed to read connection object: %+v", err)
//	return err
//}

//fmt.Printf("Received command: %s, TransactionID: %f, Object: %+v\n", commandName, transactionID, obj)

// 여기서 'connect' 요청을 처리한다고 가정합니다...

// 'connect' 요청에 대한 성공 응답을 보냅니다.
//encoder := amf.(conn) // AMF0 인코더 초기화
//response := []interface{}{
//	"_result",
//	transactionID,
//	map[string]interface{}{"level": "status", "code": "NetConnection.Connect.Success", "description": "Connection successful."},
//}
//for _, v := range response {
//	if err := encoder.Encode(v); err != nil {
//		log.Printf("Failed to encode response: %+v", err)
//		return
//	}
//}

// decoder := amf0.NewDecoder(conn)
//
//	for {
//		// 메시지 읽기
//		var data interface{}
//
//		// 데이터 디코딩
//		if err := decoder.Decode(&data); err != nil {
//			// 연결 종료 또는 읽기 에러 처리
//			if err == io.EOF {
//				log.Printf("Client closed the connection")
//				return nil
//			}
//			log.Printf("Failed to decode AMF0 data: %+v", err)
//			return err
//		}
//
//		log.Printf("Decoded AMF0 data: %+v", data)
//
//		// 읽은 메시지에 대한 간단한 처리
//		// 예: 로그 남기기
//		//log.Printf("Received message: Type=%s, Payload=%s", msg.Type, string(msg.Payload))
//		// 실제 구현에서는 msg.Type에 따라 다양한 처리를 할 수 있습니다.
//
//		// 여기에 추가적인 메시지 처리 로직을 구현할 수 있습니다.
//	}
func Handshake1(conn *Conn) error {
	// 핸드셰이크 C0 읽기
	// 0x
	/**
	네, 맞습니다. RTMP 핸드셰이크 과정에서 클라이언트(OBS)가 서버에 처음 보내는 메시지 중 하나인 C0에는 0x03 값이 포함되어 있습니다. 이 0x03 값은 RTMP 프로토콜의 버전을 나타내는 것으로, RTMP 프로토콜의 버전 3을 의미합니다.

	RTMP 프로토콜에는 여러 버전이 있을 수 있지만, 대부분의 현대적인 RTMP 구현에서는 버전 3이 널리 사용되고 있습니다. 이 버전 번호는 서버와 클라이언트가 서로 호환 가능한 프로토콜 버전을 사용하고 있는지 확인하는 데 사용됩니다.

	C0 메시지는 단 한 바이트로 구성되어 있으며, RTMP 핸드셰이크의 시작을 알리는 매우 중요한 부분입니다. 이 바이트를 통해 서버는 클라이언트가 사용하고 있는 RTMP 프로토콜의 버전을 알 수 있고, 이후의 핸드셰이크 과정(예: C1, S0, S1 등)을 그에 맞추어 진행할 수 있습니다.
	*/
	handshake := NewHandshake(conn)

	// C0 읽기
	if err := handshake.C0(); err != nil {
		closeConn(conn)
		return err
	}

	// S0 쓰기
	// 핸드셰이크 S0 쓰기 (버전 3을 사용한다고 가정)
	// 서버가 RTMP 프로토콜 버전 3을 사용하고 있음을 클라이언트에 알리는 것이며, RTMP 핸드셰이크 과정의 일부입니
	// 호환 가능한 프로토콜 버전을 사용하고 있는지 확인하는 과정
	if err := handshake.S0(); err != nil {
		closeConn(conn)
		return err
	}

	// C1 읽기
	/**
	타임스탬프: 연결이 시작된 시간을 나타내는 값입니다. 이는 일반적으로 핸드셰이크 메시지가 생성된 시간으로, 핸드셰이크 과정의 지연 시간을 계산하는 데 사용될 수 있습니다.

	랜덤 데이터: 나머지 부분은 랜덤 데이터로 채워집니다. 이 데이터는 특별한 의미를 가지지 않지만, 핸드셰이크 과정에서 일종의 "패딩" 역할을 하며, 보안성을 약간 향상시킬 수 있습니다. 특히, 랜덤 데이터는 후속 핸드셰이크 메시지(S2)에서 사용되며, 이를 통해 클라이언트와 서버 간에 데이터가 제대로 전송되었는지를 검증하는 데 사용됩니다.
	*/
	c1 := make([]byte, 1536)
	_, err := conn.rwc.Read(c1)
	fmt.Printf("Reading C1 version %v", err) // 1536
	if err != nil {
		log.Printf("Failed to read C1: %+v", err)
		conn.rwc.Close()
		return err
	}

	// S1 쓰기 (여기서는 단순화를 위해 랜덤 데이터를 사용하지 않음)
	s1 := make([]byte, 1536)
	// 고정된 값으로 타임스탬프 설정 (첫 4바이트)
	// 여기서는 임의로 0x11223344로 설정합니다.
	timestamp := uint32(0x11223344) // 0x11 = 17, 0x22 = 34, 0x33 = 51, 0x44 = 68
	binary.BigEndian.PutUint32(s1, timestamp)

	// 나머지 바이트를 임의의 값으로 채움
	// 이 값을 나중에 확인할 수 있도록, 여기서는 0xAA로 설정합니다.
	// 0xAA = 170
	payload := bytes.Repeat([]byte{0xAA}, 1536-4) // 나머지 바이트를 모두 0xAA로 채움
	copy(s1[4:], payload)

	// S1 메시지를 채우는 로직 필요 (예: 타임스탬프와 랜덤 바이트)
	a, err3 := conn.rwc.Write(s1) // 1536
	print(a)

	if err3 != nil {
		log.Printf("Failed to write S1: %+v", err)
		conn.rwc.Close()
		return err
	}

	c2 := make([]byte, 1536) // c2 : [17, 34, 51, 68, 170, 170 ....]
	_, err = conn.rwc.Read(c2)
	// c2 앞에 4바이트 타임스탬프, 이후 170으로 채워진 1532바이트
	if err != nil {
		log.Printf("Failed to read C2: %+v", err)
		conn.rwc.Close()
		return err
	}

	_, err = conn.rwc.Write(c1) // C1을 그대로 클라이언트로 보냄으로써 S2를 구성
	if err != nil {
		log.Printf("Failed to write S2: %+v", err)
		conn.rwc.Close()
		return err
	}

	return nil
}

func closeConn(conn *Conn) {
	if err := conn.rwc.Close(); err != nil {
		log.Printf("Failed to close connection: %+v", err)
		panic(err)
	}
}
