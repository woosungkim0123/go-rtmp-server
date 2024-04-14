package connect

import (
	"bytes"
	"encoding/binary"
	"log"
)

func (c *Connection) handshake() (err error) {
	if err = c.C0(); err != nil {
		return
	}
	if err = c.S0(); err != nil {
		return
	}
	if err = c.C1(); err != nil {
		return
	}
	if err = c.S1(); err != nil {
		return
	}

	return
}

// C0 RTMP 핸드셰이크의 첫 번째 바이트를 읽습니다.
// 이 바이트는 0x03이며, RTMP 프로토콜의 버전 3을 나타냅니다.
func (c *Connection) C0() (err error) {
	c0 := make([]byte, 1)
	if _, err = c.Conn.Read(c0); err != nil {
		log.Printf("Failed to read C0: %+v", err)
		return
	}
	log.Printf("Reading C0 version %v", c0[0])
	return
}

// S0 RTMP 핸드셰이크의 첫 번째 바이트를 씁니다.
// 이 바이트는 0x03이며, RTMP 프로토콜의 버전 3을 나타냅니다.
func (c *Connection) S0() (err error) {
	s0 := []byte{0x03}
	if _, err = c.Conn.Write(s0); err != nil {
		log.Printf("Failed to write S0: %+v", err)
		return
	}
	log.Printf("Writing S0 version %v", s0[0])
	return
}

// C1 타임 스탬프(4바이트) + 의미없는 제로 값(4바이트) + 랜덤 데이터 (1528바이트)를 읽습니다.
func (c *Connection) C1() (err error) {
	c1 := make([]byte, 1536)
	if _, err = c.Conn.Read(c1); err != nil {
		log.Printf("Failed to read C1: %+v", err)
		return
	}
	log.Printf("Read C1 message: timestamp=%v, zero=%v, random data length=%d", binary.BigEndian.Uint32(c1[:4]), binary.BigEndian.Uint32(c1[4:8]), len(c1[8:]))
	return
}

// S1 타임 스탬프(4바이트) + 의미없는 제로 값(4바이트) + 랜덤 데이터 (1528바이트)를 씁니다.
func (c *Connection) S1() (err error) {
	s1 := make([]byte, 1536)
	timestamp := uint32(0x11223344)
	binary.BigEndian.PutUint32(s1, timestamp)
	payload := bytes.Repeat([]byte{0xAA}, 1536-4) // 나머지 바이트를 모두 0xAA로 채움
	copy(s1[4:], payload)
	log.Printf("Write S1 message: timestamp=%v, zero=%v, random data length=%d", timestamp, binary.BigEndian.Uint32(s1[4:8]), len(s1[8:]))
	return
}

/**
RTMP (Real-Time Messaging Protocol) 핸드셰이크 과정의 C1과 S1 단계는 서버와 클라이언트 간의 연결 설정에 중요한 역할을 합니다. 이 단계들은 연결의 타이밍과 보안을 확인하고 조정하는 데 사용됩니다.

C1 (Client1)
C1은 클라이언트가 서버에 보내는 두 번째 메시지입니다. C0 메시지에서 RTMP 프로토콜 버전을 전송한 후, C1 메시지를 통해 보다 구체적인 연결 정보를 제공합니다. C1 메시지는 다음과 같은 구성 요소를 포함합니다:

타임스탬프 (4바이트): 이 타임스탬프는 클라이언트가 C1 메시지를 생성할 때의 시간을 기록합니다. 이는 네트워크 지연 측정과 동기화에 사용됩니다.
제로 (4바이트): 초기 RTMP 사양에서는 이 부분이 사용되지 않고 0으로 설정됩니다.
랜덤 데이터 (1528바이트): 이 랜덤 데이터는 보안을 강화하고 연결 중 잠재적인 공격자로부터의 위협을 방지하는 데 사용됩니다. 또한, 이 데이터는 S2 메시지 생성에 필요한 정보를 포함합니다.
S1 (Server1)
S1은 서버가 클라이언트의 C1 메시지를 받고 나서 보내는 응답 메시지입니다. S1 메시지의 구조는 C1과 매우 유사합니다:

타임스탬프 (4바이트): 서버가 S1 메시지를 생성할 때의 시간을 기록합니다.
제로 (4바이트): 이 부분도 대부분의 경우 사용되지 않고 0으로 설정됩니다.
랜덤 데이터 (1528바이트): 서버에서 생성된 랜덤 데이터로, 클라이언트와의 보안 연결을 강화하며, 이는 클라이언트가 이어서 보낼 S2 메시지의 일부가 됩니다.
C1과 S1 단계는 서로의 시스템 시간과 랜덤 데이터를 교환함으로써, 두 장치가 서로를 인증하고, 연결의 보안을 확인하며, 서로의 실시간 데이터 통신을 위한 준비를 마치게 됩니다. 이러한 정보 교환은 연결의 안전성과 신뢰성을 높이는 데 기여합니다.

블로그2
*/
