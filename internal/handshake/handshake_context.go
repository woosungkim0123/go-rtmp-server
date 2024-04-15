package handshake

type C0Data struct {
	Version uint8
}

type S0Data struct {
	Version uint8
}

type C1Data struct {
	Timestamp uint32
	Zero      uint32
	Random    []byte
}

type S1Data struct {
	Timestamp uint32
	Zero      uint32
	Random    []byte
}

type C2Data struct {
	S1Timestamp uint32
	S1Zero      uint32
	S1Random    []byte
}

type S2Data struct {
	C1Timestamp uint32
	C1Zero      uint32
	C1Random    []byte
}
