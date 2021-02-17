package uid

import "encoding/binary"

type Provider interface {
	Generate() (Identifier, error)
	MustGenerate() Identifier
	Parse(id string) (Identifier, error)
	Byte(id []byte) Identifier
	Int16(id []int16) Identifier
	Uint16(id []uint16) Identifier
	Int32(id []int32) Identifier
	Uint32(id []uint32) Identifier
	Int64(id []int64) Identifier
	Uint64(id []uint64) Identifier
}

func NewProviderCustom(size uint32, rand Randomizer, enc Encoder) Provider {
	return &Providing{
		randSize: size,
		rand:     rand,
		enc:      enc,
	}
}

func NewProvider() Provider {
	return NewProviderCustom(SizeDefault, NewRand(), NewEncoder())
}

func NewProviderSize(size uint32) Provider {
	return NewProviderCustom(size, NewRand(), NewEncoder())
}

func NewProvider36() Provider {
	return NewProviderCustom(SizeDefault, NewRand(), NewEncoderBase36())
}

func NewProvider36Size(size uint32) Provider {
	return NewProviderCustom(size, NewRand(), NewEncoderBase36())
}

func NewProvider62() Provider {
	return NewProviderCustom(SizeDefault, NewRand(), NewEncoderBase62())
}

func NewProvider62Size(size uint32) Provider {
	return NewProviderCustom(size, NewRand(), NewEncoderBase62())
}

func NewProviderUrl64() Provider {
	return NewProviderCustom(SizeDefault, NewRand(), NewEncoderBase64Url())
}

func NewProviderUrl64Size(size uint32) Provider {
	return NewProviderCustom(size, NewRand(), NewEncoderBase64Url())
}

type Providing struct {
	randSize uint32
	rand     Randomizer
	enc      Encoder
}

func (p Providing) Generate() (Identifier, error) {
	data, err := p.rand.Generate(p.randSize)
	if err != nil {
		return nil, &RandError{err}
	}

	return p.Byte(data), nil
}

// Generate random identifier or panic error
func (p Providing) MustGenerate() Identifier {
	id, err := p.Generate()
	if err != nil {
		panic(err)
	}

	return id
}

func (p Providing) Parse(id string) (Identifier, error) {
	data, err := p.enc.DecodeString(id)
	if err != nil {
		return nil, &EncError{err}
	}

	return p.Byte(data), nil
}

func (p Providing) Byte(id []byte) Identifier {
	return NewID(id, p.enc)
}

func (p Providing) Int16(id []int16) Identifier {
	uID := make([]uint16, len(id))
	for i, val := range id {
		uID[i] = uint16(val)
	}

	return p.Uint16(uID)
}

func (p Providing) Uint16(id []uint16) Identifier {
	dataSize := SizeUint16 * len(id)
	data := make([]byte, dataSize)
	for idx, val := range id {
		curIdx := idx * SizeUint16
		dstIdx := curIdx + SizeUint16
		binary.LittleEndian.PutUint16(data[curIdx:dstIdx], val)
	}

	return p.Byte(data)
}

func (p Providing) Int32(id []int32) Identifier {
	uID := make([]uint32, len(id))
	for i, val := range id {
		uID[i] = uint32(val)
	}

	return p.Uint32(uID)
}

func (p Providing) Uint32(id []uint32) Identifier {
	dataSize := SizeUint32 * len(id)
	data := make([]byte, dataSize)
	for idx, val := range id {
		curIdx := idx * SizeUint32
		dstIdx := curIdx + SizeUint32
		binary.LittleEndian.PutUint32(data[curIdx:dstIdx], val)
	}

	return p.Byte(data)
}

func (p Providing) Int64(id []int64) Identifier {
	uID := make([]uint64, len(id))
	for i, val := range id {
		uID[i] = uint64(val)
	}

	return p.Uint64(uID)
}

func (p Providing) Uint64(id []uint64) Identifier {
	dataSize := SizeUint64 * len(id)
	data := make([]byte, dataSize)
	for idx, val := range id {
		curIdx := idx * SizeUint64
		dstIdx := curIdx + SizeUint64
		binary.LittleEndian.PutUint64(data[curIdx:dstIdx], val)
	}

	return p.Byte(data)
}
