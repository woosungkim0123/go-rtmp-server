package internal

import (
	"bufio"
	"encoding/binary"
	"example/hello/internal/handshake"
	"example/hello/internal/util/endian"
	"fmt"
	"io"
	"log"
	"net"
)

type Connection struct {
	Conn              net.Conn
	Reader            *bufio.Reader
	Writer            *bufio.Writer
	ReadBuffer        []byte
	WriteBuffer       []byte
	csMap             map[uint32]*rtmpChunk
	ReadMaxChunkSize  int
	WriteMaxChunkSize int
	Context           *StreamContext

	ConnectionStatus *ConnectionStatus
}

type ConnectionStatus struct {
	HandShakeDone  bool
	ConnectionDone bool
	GotMessage     bool
}

func NewConnection(conn net.Conn, ctx *StreamContext) *Connection {
	return &Connection{
		Conn:              conn,
		Reader:            bufio.NewReader(conn),
		Writer:            bufio.NewWriter(conn),
		ReadBuffer:        make([]byte, 5096),
		WriteBuffer:       make([]byte, 5096),
		csMap:             make(map[uint32]*rtmpChunk),
		ReadMaxChunkSize:  128,
		WriteMaxChunkSize: 4096,
		Context:           ctx,
		ConnectionStatus:  &ConnectionStatus{},
	}
}

func (c *Connection) Serve() (err error) {
	if err = c.handshake(); err != nil {
		return
	}

	err = c.streamSetup()
	if err != nil {
		return
	}
	return
}

func (c *Connection) handshake() (err error) {
	err = handshake.NewHandShake(c.Conn).Handshake()
	if err != nil {
		log.Printf("Handshake failed: %s", err.Error())
		return
	}
	c.ConnectionStatus.HandShakeDone = true
	return
}

func (c *Connection) streamSetup() (err error) {
	for {
		if err = c.readMessage(); err != nil {
			return
		}
		if c.ConnectionStatus.ConnectionDone {
			return
		}
	}
}

func (c *Connection) readMessage() (err error) {
	c.ConnectionStatus.GotMessage = false
	for {
		if err = c.readChunk(); err != nil {
			fmt.Println("Error while reading message")
			return
		}
		if c.ConnectionStatus.GotMessage {
			return
		}
	}
}

func (c *Connection) readChunk() (err error) {
	var bytesRead int

	// 기본 헤더를 읽습니다. (기본적으로 1바이트)
	if _, err = io.ReadFull(c.Reader, c.ReadBuffer[:1]); err != nil {
		return
	}
	bytesRead++

	// 읽은 1바이트 중 상위 2비트는 헤더의 형식(fmt)을 나타내고 하위 6비트는 청크 스트림 ID(csID)를 나타냅니다.
	csID := uint32(c.ReadBuffer[0]) & 0x3f // 비트마스킹 연산을 통해 하위 6비트를 추출하여 csID 값을 얻습니다. (0x3f = 0011 1111)
	_fmt := uint8(c.ReadBuffer[0] >> 6)    // 6비트를 오른쪽으로 시프트하여 상위 2비트를 추출하여 fmt 값을 얻습니다.

	// 예약된 csID 값: RTMP에서는 csID 값 0과 1을 특별한 용도로 사용합니다. 0과 1은 다음에 오는 바이트(들)에서 실제 csID를 읽어야 한다는 표시로 사용됩니다.
	// RTMP 프로토콜 규칙에 따라, 0과 1은 실제 스트림 ID를 나타내는 데 사용되는 예약된 값들이기 때문에
	if csID == 0 {
		// csID가 0이라면, 실제 csID는 다음 1바이트에서 읽어야 함.
		// 실제 csID 값은 기본적으로 2 이상이어야 합니다. csID가 0인 경우, 다음 바이트에서 읽히는 값에 64를 더함으로써 64부터 319까지의 범위를 생성합니다. 이렇게 하면 한 바이트로 표현할 수 있는 값(0~255)을 이용하여 더 넓은 범위의 csID를 효율적으로 관리할 수 있습니다.
		// RTMP 프로토콜에서 csID의 값으로 0을 사용하는 것은 추가 바이트를 통해 표현할 수 있는 값의 범위를 확장하기 위한 방법입니다. 한 바이트로는 최대 255까지만 표현할 수 있으나, 이렇게 특별한 값을 사용하여 다음 바이트로부터 읽은 값을 기반으로 실제 csID를 계산함으로써, 이 범위를 넘어서는 다양한 csID를 효율적으로 할당할 수 있습니다.
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+1]); err != nil {
			return
		}
		csID = uint32(c.ReadBuffer[bytesRead]) + 64
		bytesRead++
	} else if csID == 1 {
		// csID가 1이라면, 실제 csID는 다음 2바이트에서 읽어야 합니다.
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+2]); err != nil {
			return
		}
		csID = (uint32(c.ReadBuffer[bytesRead+1]) * 256) + uint32(c.ReadBuffer[bytesRead]+64)
		bytesRead += 2
	}

	// csMap은 각 청크 스트림 ID에 대한 마지막 청크의 상태를 저장하는데 사용
	// 스트림의 연속성 유지: 스트리밍 데이터는 연속된 청크들로 전송됩니다. 각 청크는 이전 청크의 데이터에 이어지므로, 각 스트림의 마지막 상태를 저장하는 것이 중요합니다. 이를 통해 데이터가 순차적으로 올바르게 처리될 수 있습니다.
	// 스트림 관리의 효율성: 서버는 여러 csID에 대해 동시에 여러 데이터 스트림을 처리할 수 있어야 합니다. 이를 위해 각 스트림의 최신 상태를 저장하고 빠르게 접근할 수 있도록 관리하는 것이 필요합니다.
	// 오류 및 재연결 처리: 네트워크 오류나 다른 이유로 연결이 끊겼다가 재연결되는 경우, 마지막으로 처리된 청크의 상태를 기반으로 스트림을 재개할 수 있습니다.
	chunk, ok := c.csMap[csID]
	if !ok {
		log.Printf("New Chunk %d", csID)
		chunk = c.createRtmpChunk(_fmt, csID)
	}

	/*
		if fmt is 2, then message header is 3 bytes long, has only timestamp delta 이전 청크의 타임스탬프와 현재 청크의 타임스탬프 사이의 차이를 나타냅니다. 즉, 이전 청크 이후 얼마나 많은 시간이 지났는지를 밀리세컨드 단위로 표현합니다.
		if fmt is 1, then message header is 7 bytes long, has timestamp delta, message length, message type id
		if fmt is 0, then message header is 11 bytes long, has absolute timestamp, message length, message type id, message stream id
	*/

	/*
		timestamp - 		 		3 bytes
		message length - 		3 byte
		message type id- 		1 byte
		message stream id-	4 bytes, little endian

	*/

	// 3바이트 timeStamp
	if _fmt <= 2 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+3]); err != nil {
			return
		}
		chunk.header.timestamp = endian.U24BE(c.ReadBuffer[bytesRead : bytesRead+3])
		bytesRead += 3
	}

	if _fmt <= 1 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+4]); err != nil {
			return
		}
		chunk.header.length = endian.U24BE(c.ReadBuffer[bytesRead : bytesRead+3])
		chunk.header.messageType = uint8(c.ReadBuffer[bytesRead+3])
		bytesRead += 4
	}

	// TODO 왜 LITTLE EDNDIAN을 썻는가?
	if _fmt == 0 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+4]); err != nil {
			return
		}
		chunk.header.messageStreamID = binary.LittleEndian.Uint32(c.ReadBuffer[bytesRead : bytesRead+4])
		bytesRead += 4
	}
	// chunk timestamp 처리를위한 로직
	// timestamp가 다차면 0xFFFFFF로 표시되고, 4바이트 추가로 읽어들여야 함.
	if chunk.header.timestamp == 0xFFFFFF {
		chunk.header.hasExtendedTimestamp = true
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+4]); err != nil {
			return
		}
		extendedTimestamp := binary.BigEndian.Uint32(c.ReadBuffer[bytesRead : bytesRead+4])
		chunk.header.timestamp = extendedTimestamp
		bytesRead += 4
	} else {
		chunk.header.hasExtendedTimestamp = false
	}

	// MESSAGEtype20이 들어오면 AMF0 Encoded Command message
	chunk.delta = chunk.header.timestamp
	chunk.clock += chunk.header.timestamp

	// payload, capacity 초기화
	if chunk.bytes == 0 {
		chunk.payload = make([]byte, chunk.header.length)
		chunk.capacity = chunk.header.length
	}

	// 청크 데이터 전체길이, chunk.bytes는 현재까지 읽은 바이트 수
	size := int(chunk.header.length) - chunk.bytes
	if size > c.ReadMaxChunkSize {
		size = c.ReadMaxChunkSize
	}
	// 4096 시스템이 한번에 처리할수있는 최대 크기 (4KB)
	n, err := io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+size])
	if err != nil {
		return
	}
	chunk.payload = append(chunk.payload[:chunk.bytes], c.ReadBuffer[bytesRead:bytesRead+size]...)
	chunk.bytes += n
	bytesRead += n

	//if c.Stage == commandStageDone {
	//	temp := make([]byte, bytesRead)
	//	copy(temp, c.ReadBuffer[:bytesRead])
	//	if c.GotFirstAudio && c.GotFirstVideo {
	//		// c.GOP = append(c.GOP, temp)
	//	}
	//	for _, client := range c.Clients {
	//		client.Send <- temp
	//	}
	//}

	// 모든 데이터를 읽었을 때
	if chunk.bytes == int(chunk.header.length) {
		c.ConnectionStatus.GotMessage = true
		chunk.bytes = 0
		c.handleChunk(chunk)
	}

	c.csMap[chunk.header.csID] = chunk
	bytesRead = 0
	return
}

func (c *Connection) handleChunk(chunk *rtmpChunk) {
	switch chunk.header.messageType {

	case 1: // Set Max Read Chunk Size
		c.ReadMaxChunkSize = int(binary.BigEndian.Uint32(chunk.payload))

	case 5:
		// Window ACK Size

	// AMF0 Command
	//case 20:
	//	c.handleAmf0Commands(chunk)
	//
	//case 18:
	//	c.handleDataMessages(chunk)
	//
	//case 8:
	//	c.handleAudioData(chunk)
	//
	//case 9:
	//	c.handleVideoData(chunk)

	default:
		fmt.Println("UNKNOWN CHUNK RECEIVED", chunk.header.messageType)
		fmt.Println(chunk.header.fmt, chunk.header.csID, chunk.header.hasExtendedTimestamp, chunk.header.length, chunk.header.messageStreamID, chunk.header.messageType, chunk.header.timestamp)
		// panic(chunk.header.messageStreamID)
	}
}
