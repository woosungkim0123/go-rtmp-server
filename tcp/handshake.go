package tcp

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

type Handshake struct {
	rwc io.ReadWriteCloser
}

func NewHandshake(conn *Conn) *Handshake {
	return &Handshake{rwc: conn.rwc}
}

// C0 reads the first byte of the RTMP handshake
func (h *Handshake) C0() error {
	version := make([]byte, 1)
	if _, err := h.rwc.Read(version); err != nil {
		log.Printf("Failed to read C0: %+v", err)
		return err
	}
	log.Printf("Reading C0 version %v", version[0])
	return nil
}

// S0
// 0x03 =
func (h *Handshake) S0() error {
	version := []byte{0x03}
	if _, err := h.rwc.Write(version); err != nil {
		log.Printf("Failed to write S0: %+v", err)
		return err
	}
	log.Printf("Writing S0 version %v", version[0])
	return nil
}

func (h *Handshake) C1() error {
	c1 := make([]byte, 1536)
	if _, err := h.rwc.Read(c1); err != nil {
		log.Printf("Failed to read C1: %+v", err)
		return err
	}
	log.Printf("Reading C1 version %v", c1)
	return nil
}

func (h *Handshake) S1() error {
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

	//_, err := h.conn.Write(s1) // 1536
	//
	//if err3 != nil {
	//	log.Printf("Failed to write S1: %+v", err)
	//	conn.Close()
	//	continue
	//}
	return nil
}
