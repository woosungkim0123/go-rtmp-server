package internal

import "io"

type ChunkStreamerReader struct {
	reader            io.Reader
	totalReadBytes    uint32 // TODO: Check overflow
	fragmentReadBytes uint32
}

func (r *ChunkStreamerReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	r.totalReadBytes += uint32(n)
	r.fragmentReadBytes += uint32(n)
	return n, err
}
