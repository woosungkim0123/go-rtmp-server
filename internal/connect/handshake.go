package connect

import "log"

func (c *Connection) handshake() (err error) {
	if err = c.C0(); err != nil {
		return
	}

	if err = c.S0(); err != nil {
		return
	}

	return
}

// C0 reads the first byte of the RTMP handshake
func (c *Connection) C0() (err error) {
	version := make([]byte, 1)
	if _, err = c.Conn.Read(version); err != nil {
		log.Printf("Failed to read C0: %+v", err)
		return
	}
	log.Printf("Reading C0 version %v", version[0])
	return
}

func (c *Connection) S0() (err error) {
	version := []byte{0x03}
	if _, err = c.Conn.Write(version); err != nil {
		log.Printf("Failed to write S0: %+v", err)
		return
	}
	log.Printf("Writing S0 version %v", version[0])
	return
}
