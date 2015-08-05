package geoip

import (
	"fmt"
	"math"
	"math/big"
)

// For the MMDB format, see http://maxmind.github.io/MaxMind-DB/

type valueType int

const (
	typeExtended valueType = iota
	typePointer
	typeString
	typeDouble
	typeBytes
	typeUint16
	typeUint32
	typeMap
	typeInt32
	typeUint64
	typeUint128
	typeArray
	typeContainer
	typeEnd
	typeBoolean
	typeFloat
)

func decodeType(data []byte) (valueType, bool) {
	// first 3 bits
	t := data[0] >> 5
	if t == 0 {
		// extended type
		return 7 + valueType(data[1]), true
	}
	return valueType(t), false
}

func decodeSize(data []byte, extended bool) (int, int) {
	offset := 1
	if extended {
		offset++
	}
	// grab 5 lowest bits
	val := data[0] & 0x1F
	if val < 29 {
		return int(val), offset
	}
	if val == 29 {
		// 29 + next byte
		return 29 + int(data[offset]), offset + 1
	}
	if val == 30 {
		// 285 + next 2 bytes as be uint
		return 285 + int(uint32(data[offset])<<8|uint32(data[offset+1])), offset + 2
	}
	// 31 - 65821 + next 3 bytes as be uint
	return 65821 + int(uint32(data[offset])<<16|uint32(data[offset+1])<<8|uint32(data[offset+2])), offset + 3
}

func decodeUint16(data []byte, size int) uint16 {
	val := uint16(0)
	for ii := 0; ii < size; ii++ {
		val = val<<8 | uint16(data[ii])
	}
	return val
}

func decodeUint32(data []byte, size int) uint32 {
	val := uint32(0)
	for ii := 0; ii < size; ii++ {
		val = val<<8 | uint32(data[ii])
	}
	return val
}

func decodeInt32(data []byte, size int) int32 {
	return int32(decodeUint32(data, size))
}

func decodeUint64(data []byte, size int) uint64 {
	val := uint64(0)
	for ii := 0; ii < size; ii++ {
		val = val<<8 | uint64(data[ii])
	}
	return val
}

func decodeUint128(data []byte, size int) *big.Int {
	n := new(big.Int)
	return n.SetBytes(data)
}

type decoder struct {
	data []byte
	at   int
}

func (d *decoder) curData() ([]byte, error) {
	if d.at > len(d.data) {
		return nil, fmt.Errorf("invalid data pointer %d - corrupted database?", d.at)
	}
	return d.data[d.at:], nil
}

func (d *decoder) decodeType() (valueType, int, error) {
	cur, err := d.curData()
	if err != nil {
		return 0, 0, err
	}
	t, extended := decodeType(cur)
	var size, offset int
	if t == typePointer {
		// pointers look like 001SSVVV
		base := int(cur[0])
		ss := (base >> 3) & 0x03
		vvv := base & 0x07
		d.at += 2 + ss
		if ss == 0 {
			return t, vvv<<8 | int(cur[1]), nil
		}
		if ss == 1 {
			return t, (vvv<<16 | int(cur[1])<<8 | int(cur[2])) + 2048, nil
		}
		if ss == 2 {
			return t, (vvv<<24 | int(cur[1])<<16 | int(cur[2])<<8 | int(cur[3])) + 526336, nil
		}
		if ss == 3 {
			return t, int(cur[1])<<24 | int(cur[2])<<16 | int(cur[3])<<8 | int(cur[4]), nil
		}
		panic("unreachable")
	}
	size, offset = decodeSize(cur, extended)
	d.at += offset
	return t, size, nil
}

func (d *decoder) decode() (interface{}, error) {
	t, size, err := d.decodeType()
	if err != nil {
		return nil, err
	}
	cur, err := d.curData()
	if err != nil {
		return nil, err
	}
	switch t {
	case typePointer:
		dec := &decoder{d.data, size}
		return dec.decode()
	case typeString:
		d.at += size
		return makeString(cur[:size]), nil
	case typeDouble:
		if size != 8 {
			err = fmt.Errorf("double must 8 bytes, not %d", size)
			break
		}
		d.at += size
		b := decodeUint64(cur, 8)
		return math.Float64frombits(b), nil
	case typeBytes:
		d.at += size
		// Return a copy, we don't want callers
		// alterting our internal data block
		b := make([]byte, size)
		copy(b, cur)
		return b, nil
	case typeUint16:
		if size > 2 {
			err = fmt.Errorf("size %d is too big for uint16", size)
			break
		}
		d.at += size
		return decodeUint16(cur, size), nil
	case typeUint32:
		if size > 4 {
			err = fmt.Errorf("size %d is too big for uint32", size)
			break
		}
		d.at += size
		return decodeUint32(cur, size), nil
	case typeMap:
		return d.decodeMap(size)
	case typeInt32:
		if size > 4 {
			err = fmt.Errorf("size %d is too big for int32", size)
			break
		}
		d.at += size
		return decodeInt32(cur, size), nil
	case typeUint64:
		if size > 8 {
			err = fmt.Errorf("size %d is too big for uint64", size)
			break
		}
		d.at += size
		return decodeUint64(cur, size), nil
	case typeUint128:
		if size > 16 {
			err = fmt.Errorf("size %d is too big for uint128", size)
			break
		}
		d.at += size
		if size <= 8 {
			return decodeUint64(cur, size), nil
		}
		return decodeUint128(cur, size), nil
	case typeArray:
		return d.decodeArray(size)
	case typeBoolean:
		return size != 0, nil
	case typeFloat:
		if size != 4 {
			err = fmt.Errorf("float must 4 bytes, not %d", size)
			break
		}
		d.at += size
		b := decodeUint32(cur, 4)
		return math.Float32frombits(b), nil
	}
	if err == nil {
		err = fmt.Errorf("invalid data type %d", int(t))
	}
	return nil, err
}

func (d *decoder) decodeArray(count int) ([]interface{}, error) {
	values := make([]interface{}, count)
	for ii := 0; ii < count; ii++ {
		v, err := d.decode()
		if err != nil {
			return nil, err
		}
		values[ii] = v
	}
	return values, nil
}

// fast path for decoding strings from decodeMap
func (d *decoder) decodeString() (string, error) {
	t, size, err := d.decodeType()
	if err != nil {
		return "", err
	}
	if t == typePointer {
		dec := &decoder{d.data, size}
		return dec.decodeString()
	}
	if t != typeString {
		return "", fmt.Errorf("type %d is not string", t)
	}
	end := d.at + size
	s := makeString(d.data[d.at:end])
	d.at = end
	return s, nil
}

func (d *decoder) decodeMap(count int) (map[string]interface{}, error) {
	m := make(map[string]interface{}, count)
	for ii := 0; ii < count; ii++ {
		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		value, err := d.decode()
		if err != nil {
			return nil, err
		}
		m[key] = value
	}
	return m, nil
}
