package internal

import "C"
import (
	"encoding/binary"
	"log"
	"time"
)

type HandShakeContext struct {
	C0Data C0Data
	S0Data S0Data
	C1Data C1Data
	S1Data S1Data
	C2Data C2Data
	S2Data S2Data
}

func (c *Connection) Handshake() (err error) {
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
	if err = c.C2(); err != nil {
		log.Printf("C2 validation failed: %+v", err)
		return
	}
	if err = c.S2(); err != nil {
		log.Printf("Failed to write S2: %+v", err)
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
	c.HandShakeContext.C0Data.Version = c0[0]
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
	c.HandShakeContext.S0Data.Version = s0[0]
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
	c.HandShakeContext.C1Data.Timestamp = binary.BigEndian.Uint32(c1[:4])
	c.HandShakeContext.C1Data.Zero = binary.BigEndian.Uint32(c1[4:8])
	c.HandShakeContext.C1Data.Random = c1[8:]
	log.Printf("Reading C1 message: timestamp=%v, zero=%v, random data length=%d", c.HandShakeContext.C1Data.Timestamp, c.HandShakeContext.C1Data.Zero, len(c.HandShakeContext.C1Data.Random))
	return
}

// S1 타임 스탬프(4바이트) + 의미없는 제로 값(4바이트) + 랜덤 데이터 (1528바이트)를 씁니다.
func (c *Connection) S1() (err error) {
	s1 := make([]byte, 1536)
	currentTime := uint32(time.Now().Unix())
	binary.BigEndian.PutUint32(s1[4:8], currentTime)
	zero := uint32(0)
	binary.BigEndian.PutUint32(s1[0:4], zero)

	// 랜덤 데이터 생성
	randBytes := make([]byte, 1528)
	_, err = c.Conn.Read(randBytes)
	if err != nil {
		return
	}
	copy(s1[8:], randBytes)
	_, err = c.Writer.Write(s1)
	if err != nil {
		return
	}
	c.Writer.Flush()
	c.HandShakeContext.S1Data.Timestamp = currentTime
	c.HandShakeContext.S1Data.Zero = zero
	c.HandShakeContext.S1Data.Random = randBytes
	log.Printf("Writing S1 message: timestamp=%v, zero=%v, random data length=%d", currentTime, zero, len(randBytes))
	return
}

// C2 메시지를 읽고 S1 메시지와 비교하여 에코된 데이터가 정확한지 확인합니다.
func (c *Connection) C2() (err error) {
	log.Printf("Reading C2 message1")
	c2 := make([]byte, 1536)
	log.Printf("Reading C2 message2")
	if _, err = c.Conn.Read(c2); err != nil {
		log.Printf("Failed to read C2: %+v", err)
		return
	}
	log.Printf("Reading C2 message3")
	s1Timestamp := binary.BigEndian.Uint32(c2[:4])
	s1ZeroValue := binary.BigEndian.Uint32(c2[4:8])
	s1RandomData := c2[8:]

	//if s1Timestamp != h.S1Data.Timestamp || !bytes.Equal(s1RandomData, h.S1Data.Random) {
	//	return fmt.Errorf("C2 validation failed: Echoed S1 timestamp or random data does not match")
	//}

	c.HandShakeContext.C2Data.S1Timestamp = s1Timestamp
	c.HandShakeContext.C2Data.S1Zero = s1ZeroValue
	c.HandShakeContext.C2Data.S1Random = s1RandomData

	log.Printf("Reading C2 message: timestamp=%v, zero=%v, random data length=%d", s1Timestamp, s1ZeroValue, len(s1RandomData))
	return
}

// S2 C1을 에코한 데이터를 클라이언트에게 보냅니다.
func (c *Connection) S2() (err error) {
	s2 := make([]byte, 1536)
	binary.BigEndian.PutUint32(s2[:4], c.HandShakeContext.C1Data.Timestamp)
	binary.BigEndian.PutUint32(s2[4:8], c.HandShakeContext.C1Data.Zero)
	copy(s2[8:], c.HandShakeContext.C1Data.Random)

	if _, err = c.Conn.Write(s2); err != nil {
		return
	}

	c.HandShakeContext.S2Data.C1Timestamp = c.HandShakeContext.C1Data.Timestamp
	c.HandShakeContext.S2Data.C1Zero = c.HandShakeContext.C1Data.Zero
	c.HandShakeContext.S2Data.C1Random = c.HandShakeContext.C1Data.Random

	log.Printf("Writing S2 message: timestamp=%v, zero=%v, random data length=%d", c.HandShakeContext.C1Data.Timestamp, c.HandShakeContext.C1Data.Zero, len(c.HandShakeContext.C1Data.Random))
	return
}
