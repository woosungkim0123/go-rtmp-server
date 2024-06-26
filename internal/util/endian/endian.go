package endian

/*
	Big Endian
	데이터의 가장 상위 비트(가장 왼쪽 비트)가 먼저 나타나며, 이어서 하위 비트가 나타나는 방식을 빅 엔디안이라고 합니다.
	네트워크에서 데이터를 주고받거나 다른 시스템 간에 데이터를 교환할 때 일반적으로 사용되는 방식 중 하나입니다.
*/

// U24BE 함수는 3바이트 크기의 빅엔디안 방식으로 24비트 부호 없는 정수를 바이트 슬라이스에 저장합니다.
func U24BE(b []byte) (i uint32) {
	i = uint32(b[0])
	i <<= 8
	i |= uint32(b[1])
	i <<= 8
	i |= uint32(b[2])
	return
}

// PutU16BE 함수는 2바이트 크기의 빅엔디안 방식으로 16비트 부호 없는 정수를 바이트 슬라이스에 저장합니다.
func PutU16BE(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

// PutU24BE 함수는 3바이트 크기의 빅엔디안 방식으로 24비트 부호 없는 정수를 바이트 슬라이스에 저장합니다.
func PutU24BE(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

// PutU32BE 함수는 4바이트 크기의 빅엔디안 방식으로 32비트 부호 없는 정수를 바이트 슬라이스에 저장합니다.
func PutU32BE(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// PutU64BE 함수는 8바이트 크기의 빅엔디안 방식으로 64비트 부호 없는 정수를 바이트 슬라이스에 저장합니다.
func PutU64BE(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}
