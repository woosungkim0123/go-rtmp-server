package internal

type StreamControlState struct {
	chunkSize           uint32
	ackWindowSize       int32
	bandwidthWindowSize int32
}
