package internal

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

type Conn struct {
	rwc      io.ReadWriteCloser
	bReader  *bufio.Reader
	bWriter  *bufio.Writer
	streamer *ChunkStreamer
	streams  *Streams
}

func NewConn(rwc io.ReadWriteCloser) *Conn {
	conn := &Conn{
		rwc:     rwc,
		bReader: bufio.NewReaderSize(rwc, 4*1024), // 4KB (Default)
		bWriter: bufio.NewWriterSize(rwc, 4*1024), // 4KB (Default)
	}
	conn.streamer = NewChunkStreamer(conn.bReader, conn.bWriter)
	conn.streams = NewStreams(conn)
	return conn
}

func (c *Conn) runHandleMessageLoop() error {
	var cmsg ChunkMessage // 하나의 RTMP 청크
	for {
		log.Println("Reading chunk")
		chunkStreamID, timestamp, err := c.streamer.Read(&cmsg)
		log.Printf("chunkStreamID: %d, timestamp: %d, err: %v", chunkStreamID, timestamp, err)
		if err != nil {
			return err
		}

		if err := c.handleMessage(chunkStreamID, timestamp, &cmsg); err != nil {
			return err // Shutdown the connection
		}
	}
}

func (c *Conn) handleMessage(chunkStreamID int, timestamp uint32, cmsg *ChunkMessage) error {
	stream, err := c.streams.At(cmsg.StreamID)
	if err != nil {
		return fmt.Errorf("Failed to get stream: Err = %+v", err)
	}

	if err = stream.handler.Handle(chunkStreamID, timestamp, cmsg.Message); err != nil {
		return fmt.Errorf("Failed to handle message: Err = %+v", err)
	}

	return nil
}
