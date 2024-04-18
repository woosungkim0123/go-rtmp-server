package flvio

import (
	"example/hello/internal/util/endian"
	"math"
	"time"
)

// Flash Video File Format (FLV) I/O

type AMFMap map[string]interface{}
type AMFArray []interface{}
type AMFECMAArray map[string]interface{}

// 타입 마커 (1바이트) + 숫자 (8바이트) = 9바이트
// 타입 마커 - 각 데이터 타입을 식별하기 위한 마커 입니다.
// 데이터 값 - 숫자의 실제 값을 IEEE 754 형식의 부동소수점 숫자로 표현합니다.
const lenAMF0Number = 9

const (
	numberMarker = iota
	booleanmarker
	stringmarker
	objectmarker
	movieclipmarker
	nullmarker
	undefinedmarker
	referencemarker
	ecmaarraymarker
	objectendmarker
	strictarraymarker
	datemarker
	longstringmarker
	unsupportedmarker
	recordsetmarker
	xmldocumentmarker
	typedobjectmarker
	avmplusobjectmarker
)

// LenAMF0Val AMF0 (Action Message Format 0) 인코딩 시 필요한 바이트 크기를 계산하는 용도로 사용됩니다.
func LenAMF0Val(_val interface{}) (n int) {
	switch val := _val.(type) {
	case int8, int16, int32, int64, int, uint8, uint16, uint32, uint64, uint, float32, float64:
		n += lenAMF0Number

	case string:
		u := len(val)
		if u <= 65536 {
			n += 3
		} else {
			n += 5
		}
		n += int(u)

	case AMFECMAArray:
		n += 5
		n += lenAMFMap(val)
		n += 3

	case AMFMap:
		n++
		n += lenAMFMap(val)
		n += 3

	case AMFArray:
		n += 5
		for _, v := range val {
			n += LenAMF0Val(v)
		}

	case time.Time:
		n += 1 + 8 + 2

	case bool:
		n += 2

	case nil:
		n++
	}

	return
}

func lenAMFMap(m map[string]interface{}) int {
	var total int
	for k, v := range m {
		if len(k) > 0 {
			total += 2 + len(k)
			total += LenAMF0Val(v)
		}
	}
	return total
}

// FillAMF0Val AMF0 (Action Message Format 0) 형식으로 값을 인코딩합니다.
func FillAMF0Val(b []byte, _val interface{}) (n int) {
	switch val := _val.(type) {
	case int8:
		n += fillAMF0Number(b[n:], float64(val))
	case int16:
		n += fillAMF0Number(b[n:], float64(val))
	case int32:
		n += fillAMF0Number(b[n:], float64(val))
	case int64:
		n += fillAMF0Number(b[n:], float64(val))
	case int:
		n += fillAMF0Number(b[n:], float64(val))
	case uint8:
		n += fillAMF0Number(b[n:], float64(val))
	case uint16:
		n += fillAMF0Number(b[n:], float64(val))
	case uint32:
		n += fillAMF0Number(b[n:], float64(val))
	case uint64:
		n += fillAMF0Number(b[n:], float64(val))
	case uint:
		n += fillAMF0Number(b[n:], float64(val))
	case float32:
		n += fillAMF0Number(b[n:], float64(val))
	case float64:
		n += fillAMF0Number(b[n:], float64(val))

	case string:
		u := len(val)
		if u <= 65536 {
			b[n] = stringmarker
			n++
			endian.PutU16BE(b[n:], uint16(u))
			n += 2
		} else {
			b[n] = longstringmarker
			n++
			endian.PutU32BE(b[n:], uint32(u))
			n += 4
		}
		copy(b[n:], []byte(val))
		n += len(val)

	case AMFECMAArray:
		b[n] = ecmaarraymarker
		n++
		endian.PutU32BE(b[n:], uint32(len(val)))
		n += 4
		for k, v := range val {
			endian.PutU16BE(b[n:], uint16(len(k)))
			n += 2
			copy(b[n:], []byte(k))
			n += len(k)
			n += FillAMF0Val(b[n:], v)
		}
		endian.PutU24BE(b[n:], 0x000009)
		n += 3

	case AMFMap:
		b[n] = objectmarker
		n++
		for k, v := range val {
			if len(k) > 0 {
				endian.PutU16BE(b[n:], uint16(len(k)))
				n += 2
				copy(b[n:], []byte(k))
				n += len(k)
				n += FillAMF0Val(b[n:], v)
			}
		}
		endian.PutU24BE(b[n:], 0x000009)
		n += 3

	case AMFArray:
		b[n] = strictarraymarker
		n++
		endian.PutU32BE(b[n:], uint32(len(val)))
		n += 4
		for _, v := range val {
			n += FillAMF0Val(b[n:], v)
		}

	case time.Time:
		b[n] = datemarker
		n++
		u := val.UnixNano()
		f := float64(u / 1000000)
		n += fillBEFloat64(b[n:], f)
		endian.PutU16BE(b[n:], uint16(0))
		n += 2

	case bool:
		b[n] = booleanmarker
		n++
		var u uint8
		if val {
			u = 1
		} else {
			u = 0
		}
		b[n] = u
		n++

	case nil:
		b[n] = nullmarker
		n++
	}

	return
}

func fillAMF0Number(b []byte, f float64) int {
	b[0] = numberMarker
	fillBEFloat64(b[1:], f)
	return lenAMF0Number
}

func fillBEFloat64(b []byte, f float64) int {
	endian.PutU64BE(b, math.Float64bits(f))
	return 8
}
