package uid

import (
	"encoding/binary"
	"errors"
)

const ErrorMsgSizeDivisible2 = "size of data must be divisible by 2"
const ErrorMsgSizeDivisible4 = "size of data must be divisible by 4"
const ErrorMsgSizeDivisible8 = "size of data must be divisible by 8"

type Identifier interface {
	String() string
	Byte() []byte
	Int16() ([]int16, error)
	Uint16() ([]uint16, error)
	Int32() ([]int32, error)
	Uint32() ([]uint32, error)
	Int64() ([]int64, error)
	Uint64() ([]uint64, error)
}

type ID struct {
	data []byte
	enc  Encoder
}

func NewID(data []byte, enc Encoder) Identifier {
	return &ID{
		data: data,
		enc:  enc,
	}
}

func NewIDStdBase32(data []byte) Identifier {
	return NewID(data, NewEncoder())
}

func (id ID) String() string {
	return id.enc.EncodeToString(id.data)
}

func (id ID) Byte() []byte {
	return id.data
}

func (id ID) Int16() ([]int16, error) {
	if len(id.data)%SizeInt16 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible2)}
	}

	size := sizeNewArray(id.data, SizeInt16)
	dst := make([]int16, size)
	bs := readBytes(id.data, size, SizeInt16)
	for i, b := range bs {
		dst[i] = int16(binary.LittleEndian.Uint16(b))
	}

	return dst, nil
}

func (id ID) Uint16() ([]uint16, error) {
	if len(id.data)%SizeUint16 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible2)}
	}

	size := sizeNewArray(id.data, SizeUint16)
	dst := make([]uint16, size)
	bs := readBytes(id.data, size, SizeUint16)
	for i, b := range bs {
		dst[i] = binary.LittleEndian.Uint16(b)
	}

	return dst, nil
}

func (id ID) Int32() ([]int32, error) {
	if len(id.data)%SizeInt32 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible4)}
	}

	size := sizeNewArray(id.data, SizeInt32)
	dst := make([]int32, size)
	bs := readBytes(id.data, size, SizeInt32)
	for i, b := range bs {
		dst[i] = int32(binary.LittleEndian.Uint32(b))
	}

	return dst, nil
}

func (id ID) Uint32() ([]uint32, error) {
	if len(id.data)%SizeUint32 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible4)}
	}

	size := sizeNewArray(id.data, SizeUint32)
	dst := make([]uint32, size)
	bs := readBytes(id.data, size, SizeUint32)
	for i, b := range bs {
		dst[i] = binary.LittleEndian.Uint32(b)
	}

	return dst, nil
}

func (id ID) Int64() ([]int64, error) {
	if len(id.data)%SizeInt64 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible8)}
	}

	size := sizeNewArray(id.data, SizeInt64)
	dst := make([]int64, size)
	bs := readBytes(id.data, size, SizeInt64)
	for i, b := range bs {
		dst[i] = int64(binary.LittleEndian.Uint64(b))
	}

	return dst, nil
}

func (id ID) Uint64() ([]uint64, error) {
	if len(id.data)%SizeUint64 != 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgSizeDivisible8)}
	}

	size := sizeNewArray(id.data, SizeUint64)
	dst := make([]uint64, size)
	bs := readBytes(id.data, size, SizeInt64)
	for i, b := range bs {
		dst[i] = binary.LittleEndian.Uint64(b)
	}

	return dst, nil
}

func sizeNewArray(src []byte, size int) int {
	res := len(src) / size
	if len(src)%size != 0 {
		res += 1
	}
	return res
}

func readBytes(src []byte, dstParts int, dstSize int) [][]byte {
	if len(src) == 0 || len(src)%dstSize != 0 {
		return [][]byte{}
	}

	bs := make([][]byte, 0, dstSize)
	for i := 0; i < dstParts; i++ {
		start := i * dstSize
		end := (i + 1) * dstSize
		bs = append(bs, src[start:end])
	}

	return bs
}
