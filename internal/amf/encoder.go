package amf

import "example/hello/internal/format/flvio"

func Encode(args ...interface{}) ([]byte, int) {
	var length int
	for _, val := range args {
		length += flvio.LenAMF0Val(val)
	}
	res := make([]byte, length)
	var offset int
	for _, val := range args {
		offset += flvio.FillAMF0Val(res[offset:], val)
	}

	return res, length
}
