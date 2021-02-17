package uid

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"math/big"
)

type Encoder interface {
	EncodeToString(src []byte) string
	DecodeString(s string) ([]byte, error)
}

func NewEncoder() Encoder {
	return NewEncoderBase32Std()
}

func NewEncoderBase16Std() Encoder {
	return NewEncoderCustom(&Base16Encoding{})
}

func NewEncoderBase32Std() Encoder {
	return NewEncoderCustom(base32.StdEncoding.WithPadding(base32.NoPadding))
}

func NewEncoderBase32Hex() Encoder {
	return NewEncoderCustom(base32.HexEncoding.WithPadding(base32.NoPadding))
}

func NewEncoderBase36() Encoder {
	return NewEncoderCustom(&BaseXEncoding{base: 36})
}

func NewEncoderBaseX(base int) Encoder {
	return NewEncoderCustom(&BaseXEncoding{base: base})
}

func NewEncoderBase62() Encoder {
	return NewEncoderCustom(&BaseXEncoding{base: 62})
}

func NewEncoderBase64Std() Encoder {
	return NewEncoderCustom(base64.RawStdEncoding)
}

func NewEncoderBase64Url() Encoder {
	return NewEncoderCustom(base64.RawURLEncoding)
}

func NewEncoderCustom(encoder Encoder) Encoder {
	return &Encoding{
		Encoder: encoder,
	}
}

type Encoding struct {
	Encoder
}

type Base16Encoding struct{}

func (b Base16Encoding) EncodeToString(src []byte) string {
	return hex.EncodeToString(src)
}

func (b Base16Encoding) DecodeString(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

type BaseXEncoding struct {
	base int
}

func (b BaseXEncoding) EncodeToString(src []byte) string {
	bi := new(big.Int)
	bi.SetBytes(src)

	return bi.Text(b.base)
}

func (b BaseXEncoding) DecodeString(s string) ([]byte, error) {
	bi := new(big.Int)
	bi.SetString(s, b.base)

	return bi.Bytes(), nil
}

type MockEncoding struct {
	EncodedString string
	DecodedBytes  []byte
	DecodedErr    error
}

func (e MockEncoding) EncodeToString(src []byte) string {
	return e.EncodedString
}

func (e MockEncoding) DecodeString(s string) ([]byte, error) {
	if e.DecodedErr != nil {
		return nil, e.DecodedErr
	}

	return e.DecodedBytes, nil
}
