package uid

import (
	cryptRand "crypto/rand"
	"errors"
	"io"
)

const ErrorMsgWrongSizeZero = "size cannot be zero"

type Randomizer interface {
	Generate(size uint32) ([]byte, error)
}

type Rand struct {
	reader io.Reader
}

func NewRand() Randomizer {
	return &Rand{
		reader: &CryptRandReader{},
	}
}

func NewRandCustom(r io.Reader) Randomizer {
	return &Rand{
		reader: r,
	}
}

func (r Rand) Generate(size uint32) ([]byte, error) {
	if size == 0 {
		return nil, &WrongSizeError{errors.New(ErrorMsgWrongSizeZero)}
	}

	b := make([]byte, size)
	_, err := r.reader.Read(b)
	if err != nil {
		return nil, &InternalError{err}
	}

	return b, nil
}

type CryptRandReader struct{}

func (r CryptRandReader) Read(b []byte) (n int, err error) {
	return cryptRand.Read(b)
}

type MockRandReader struct {
	Actual    []byte
	ActualErr error
}

func (r MockRandReader) Read(b []byte) (n int, err error) {
	if r.ActualErr != nil {
		return 0, r.ActualErr
	}

	for i := range b {
		b[i] = r.Actual[i]
	}
	return len(r.Actual), nil
}
