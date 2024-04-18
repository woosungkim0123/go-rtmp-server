package internal

import (
	"bufio"
	"encoding/binary"
	"example/hello/internal/amf"
	"example/hello/internal/format/flvio"
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

	AppName string

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
		ReadBuffer:        make([]byte, 5096), // 일반적인 패킷 크기는 1500이지만 5KB 크기의 버퍼로 설정함으로써 I/O 호출을 줄이고 성능을 향상 시킬 수 있습니다.
		WriteBuffer:       make([]byte, 5096),
		csMap:             make(map[uint32]*rtmpChunk),
		ReadMaxChunkSize:  128, // 실시간 스트리밍을 위해 작은 청크 크기를 사용하여 지연을 최소화함으로써 빠른 데이터 처리 및 전송을 가능하게 하고, 최적의 사용자 경험을 제공합니다.
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

// readChunk Chunk 단위로 데이터를 읽어들입니다.
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

	if csID == 0 {
		// csID가 0이라면, 실제 csID는 다음 1바이트에서 읽어야 합니다.
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

	// csMap 은 각 청크 스트림 ID에 대한 마지막 청크의 상태를 저장하는데 사용
	chunk, ok := c.csMap[csID]
	if !ok {
		log.Printf("New Chunk %d", csID)
		chunk = c.createRtmpChunk(_fmt, csID)
	}

	// timestamp - 3 bytes
	// fmt 0 - absolute timestamp, fmt 1, 2 - timestamp delta
	if _fmt <= 2 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+3]); err != nil {
			return
		}
		chunk.header.timestamp = endian.U24BE(c.ReadBuffer[bytesRead : bytesRead+3])
		bytesRead += 3
	}

	// message length - 3 bytes, message type ID - 1 byte
	if _fmt <= 1 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+4]); err != nil {
			return
		}
		chunk.header.length = endian.U24BE(c.ReadBuffer[bytesRead : bytesRead+3])
		chunk.header.messageType = uint8(c.ReadBuffer[bytesRead+3])
		bytesRead += 4
	}

	// message stream ID - 4 bytes, little endian
	if _fmt == 0 {
		if _, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+4]); err != nil {
			return
		}
		chunk.header.messageStreamID = binary.LittleEndian.Uint32(c.ReadBuffer[bytesRead : bytesRead+4])
		bytesRead += 4
	}

	// timestamp가 다차면 0xFFFFFF로 표시되고, 4바이트 추가로 읽습니다.
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

	chunk.delta = chunk.header.timestamp  // chunk 간 시간 간격
	chunk.clock += chunk.header.timestamp // stream 내에서 현재까지의 총 진행 시간
	log.Printf("delta timestamp: %d, clock: %d", chunk.delta, chunk.clock)

	// 첫 번째 데이터를 읽을 때 payload를 초기화합니다.
	if chunk.bytes == 0 {
		chunk.payload = make([]byte, chunk.header.length)
		chunk.capacity = chunk.header.length
	}

	// 총 읽어야할 데이터량 - 현재까지 읽은 데이터량
	// 만약 읽어야할 데이터량이 한번에 읽을 수 있는 최대 읽기 크기보다 크다면 최대 읽기 크기로 설정합니다.
	size := int(chunk.header.length) - chunk.bytes
	log.Printf("chunk length: %d, bytes: %d, size: %d", chunk.header.length, chunk.bytes, size)
	if size > c.ReadMaxChunkSize {
		size = c.ReadMaxChunkSize
	}
	log.Printf("size: %d", size)

	var n int
	n, err = io.ReadFull(c.Reader, c.ReadBuffer[bytesRead:bytesRead+size])

	chunk.payload = append(chunk.payload[:chunk.bytes], c.ReadBuffer[bytesRead:bytesRead+size]...) // ...는 슬라이스의 요소를 개별적으로 풀어서 append 함수의 인자로 넘깁니다.
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
		c.ReadMaxChunkSize = int(binary.BigEndian.Uint32(chunk.payload)) // 4096

	//case 5:
	// Window ACK Size

	case 20: // AMF0 Command
		c.handleAmf0Commands(chunk)
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

func (c *Connection) handleAmf0Commands(chunk *rtmpChunk) {
	command := amf.Decode(chunk.payload)

	switch command["cmd"] {
	case "connect":
		c.onConnect(command)
	case "releaseStream":
		//c.onRelease(command)
	case "FCPublish":
		//c.onFCPublish(command)
	case "createStream":
		//c.onCreateStream(command)
	case "publish":
		//c.onPublish(command, chunk.header.messageStreamID)
	case "play":
		//c.onPlay(command, chunk)
	case "pause":
		fmt.Println("Pause")
	case "FCUnpublish":
	case "deleteStream":
		fmt.Println("Delete Stream")
	case "closeStream":
		fmt.Println("Close Stream")
	case "receiveAudio":
		fmt.Println("Receive Audio")
	case "receiveVideo":
		fmt.Println("Receive Video")
	default:
		fmt.Println("UNKNOWN AMF COMMAND RECEIVED")
	}
}

func (c *Connection) onConnect(connectCommand map[string]interface{}) {
	log.Printf("Connect Command: %v", connectCommand)

	c.AppName = connectCommand["cmdObj"].(map[string]interface{})["app"].(string)
	c.setMaxWriteChunkSize(128)
	c.sendWindowACK(5000000) // 윈도우 크기는 서버가 클라이언트로부터 얼마나 많은 데이터를 받아들일 수 있는지를 정하는 한계 값입니다. 서버가 클라이언트로부터 데이터를 받아들이는 속도를 조절하는데 사용됩니다. (5MB)

	// 대역폭은 네트워크에서 사용 가능한 최대 전송 속도를 나타냅니다.
	// RTMP 경우, 대역폭 설정은 클라이언트와 서버 간의 통신을 최적화하고 스트리밍의 품질과 안정성을 유지하기 위해 중요합니다.
	// 대역폭 값인 5000000은 5000000 바이트/초 또는 약 5 Mbps를 나타내고, 일반적으로 고화질 또는 고속 스트리밍에 적합한 대역폭 수준입니다.
	// 필요한 대역폭이나 최적의 값은 특정 상황에 따라 다를 수 있으므로, 실제 테스트 및 성능 모니터링을 통해 적절한 값을 결정하는 것이 중요합니다.
	c.setPeerBandwidth(5000000, 2)
	c.Writer.Flush()

	cmd := "_result"
	transID := connectCommand["transId"]
	cmdObj := flvio.AMFMap{
		"fmsVer":       "FMS/3,0,1,123",
		"capabilities": 31,
	}
	info := flvio.AMFMap{
		"level":          "status",
		"code":           "NetConnection.Connect.Success",
		"description":    "Connection succeeded",
		"objectEncoding": 0,
	}
	amfPayload, length := amf.Encode(cmd, transID, cmdObj, info)

	chunk := &rtmpChunk{
		header: &chunkHeader{
			fmt:             0,
			csID:            3,
			messageType:     20,
			messageStreamID: 0,
			timestamp:       0,
			length:          uint32(length),
		},
		clock:    0,
		delta:    0,
		capacity: 0,
		bytes:    0,
		payload:  amfPayload,
	}
	for _, ch := range c.create(chunk) {
		c.Writer.Write(ch)
	}
	c.Writer.Flush()

	c.ConnectionStatus.ConnectionDone = true
}
