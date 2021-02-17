# Unique ID Generator

* [Motivation](#motivation)
* [Quick Architecture Review](#quick-architecture-review)
* [Examples](#examples)
* [Contributing](#contributing)

[![Сodecov](https://codecov.io/gh/aohorodnyk/uid/branch/main/graph/badge.svg?token=TAS3HLHPWB)](https://codecov.io/gh/aohorodnyk/uid) ![Test](https://github.com/aohorodnyk/uid/workflows/Test/badge.svg) ![golangci-lint](https://github.com/aohorodnyk/uid/workflows/golangci-lint/badge.svg)

## Motivation
I had an experience with UUID, and it's perfect for various use cases. For some number of projects I have few additional requirements which UUID doesn't meet:
1. Custom size for random hash
1. Custom string encoder
1. A unique string should achieve security requirements
1. Should be readable for business code generation
1. String representation should be shorter than UUID

How requirements achieved:
1. By default, used `crypto/rand` implementation, but it extendable and can be simply changed to any other implementation (`math/rand` or any custom one)
1. By default, used base32, which is readable (because of the list of supported symbols), but can be simply replaced to any another implementation
1. By default, used base32, as a result, 20 random bytes (160 bits) are encoded to 32 symbols (UUID has 122 random bits in 32-36 symbols)
1. Supported all main base encoders: base16, base32, base36, base62, base64, and all supported by bin int

## Quick Architecture Review
The main endpoint which should be used from external applications or libraries is located in the file `provider.go`.
The file can be found functions to create default or custom providers.
`Provider` is a struct that can generate new ID (random number of bytes) or parse from a string, decode from ints, initialize from bytes ID.
`ID` is a struct that can encode or dump the id (random number of bytes) to a string, bytes, ints.
`Encoder`, `reader`, `randomizer` are internal entities which use for:
* `Encoder` - interface to encode and decode an array of bytes to a string
* `Randomizer` - interface to generate a random array of bytes, currently implemented `encoder` only from `crypto/rand` which has a secure random implementation
* `Reader` - struct implements `io.Reader` to generate rand from packages `crypto/rand` and can be implemented from `math/rand` 

## Examples

### Default Provider
In the file `provider.go` can be found a list of main exported functions for the project.
The default provider has random bytes size `20`, randomizer from `crypto/rand`, the encoder is `base32` with disabled paddings.

#### Generate random base32 string with default provider
```go
package main

import (
	"fmt"
	"github.com/aohorodnyk/uid"
)

func main() {
	p := uid.NewProvider()
	g := p.MustGenerate()

	fmt.Println(g.String()) // WY5WHCHAHISDRI35UOHTQ3ZS4THJRMP3
	fmt.Println(g.Byte()) // [182 59 99 136 224 58 36 56 163 125 163 143 56 111 50 228 206 152 177 251]
	fmt.Println(g.Int16()) // [15286 -30621 15072 14372 32163 -28765 28472 -7118 -26418 -1103]
	fmt.Println(g.Uint16()) // [15286 34915 15072 14372 32163 36771 28472 58418 39118 64433]
	fmt.Println(g.Int32()) // [-2006762570 941898464 -1885110877 -466456776 -72247090]
	fmt.Println(g.Uint32()) // [2288204726 941898464 2409856419 3828510520 4222720206]
	fmt.Println(g.Int64()) // err: size of data must be divisible by 8
	fmt.Println(g.Uint64()) // err: size of data must be divisible by 8
}
```

#### Parse ID from previously generated base32 string
```go
package main

import (
	"fmt"
	"github.com/aohorodnyk/uid"
	"log"
)

func main() {
	p := uid.NewProvider()
	g, err := p.Parse("WY5WHCHAHISDRI35UOHTQ3ZS4THJRMP3")
	if err != nil {
		log.Panicln("Cannot parse random string")
	}

	fmt.Println(g.String()) // WY5WHCHAHISDRI35UOHTQ3ZS4THJRMP3
	fmt.Println(g.Byte()) // [182 59 99 136 224 58 36 56 163 125 163 143 56 111 50 228 206 152 177 251]
	fmt.Println(g.Int16()) // [15286 -30621 15072 14372 32163 -28765 28472 -7118 -26418 -1103]
	fmt.Println(g.Uint16()) // [15286 34915 15072 14372 32163 36771 28472 58418 39118 64433]
	fmt.Println(g.Int32()) // [-2006762570 941898464 -1885110877 -466456776 -72247090]
	fmt.Println(g.Uint32()) // [2288204726 941898464 2409856419 3828510520 4222720206]
	fmt.Println(g.Int64()) // err: size of data must be divisible by 8
	fmt.Println(g.Uint64()) // err: size of data must be divisible by 8
}
```

#### Generate random base32 string with custom size
```go
package main

import (
	"fmt"
	"github.com/aohorodnyk/uid"
)

func main() {
	p := uid.NewProviderSize(32)
	g := p.MustGenerate()

	fmt.Println(g.String()) // 3C4SZMODOFHXKDFPRZBBEEMBDB22DUL2A6TPOZXAHDAJJO56PGGQ
	fmt.Println(g.Byte()) // [216 185 44 177 195 113 79 117 12 175 142 66 18 17 129 24 117 161 209 122 7 166 247 102 224 56 192 148 187 190 121 141]
	fmt.Println(g.Int16()) // [-17960 -20180 29123 30031 -20724 17038 4370 6273 -24203 31441 -23033 26359 14560 -27456 -16709 -29319]
	fmt.Println(g.Uint16()) // [47576 45356 29123 30031 44812 17038 4370 6273 41333 31441 42503 26359 14560 38080 48827 36217]
	fmt.Println(g.Int32()) // [-1322468904 1968140739 1116647180 411111698 2060558709 1727505927 -1799341856 -1921401157]
	fmt.Println(g.Uint32()) // [2972498392 1968140739 1116647180 411111698 2060558709 1727505927 2495625440 2373566139]
	fmt.Println(g.Int64()) // [8453100110902770136 1765711299029675788 7419581462171722101 -8252355129315936032]
	fmt.Println(g.Uint64()) // [8453100110902770136 1765711299029675788 7419581462171722101 10194388944393615584]
}
```

To parse the string can be used previous example. `Size` parameter needed only for random ID generation and ignored in all other methods.

### Custom Base62 encoder
All other encoders can be used in the same way.

#### Generate random base62 string
```go
package main

import (
	"fmt"
	"github.com/aohorodnyk/uid"
)

func main() {
	// Can be used as alternative:
	// * uid.NewProvider62Size(8)
	// * uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBase62())
	// * uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBaseX(62))
	p := uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBase62())
	g := p.MustGenerate()

	fmt.Println(g.String()) // 8twXZ4Nkui7
	fmt.Println(g.Byte()) // [98 186 154 157 247 107 100 139] // 8 bytes
	fmt.Println(g.Int16()) // [-17822 -25190 27639 -29852] // 4 bytes
	fmt.Println(g.Uint16()) // [47714 40346 27639 35684] // 4 bytes
	fmt.Println(g.Int32()) // [-1650804126 -1956353033] // 2 bytes
	fmt.Println(g.Uint32()) // [2644163170 2338614263] // 2 bytes
	fmt.Println(g.Int64()) // [-8402472293521245598] // 1 byte
	fmt.Println(g.Uint64()) // [10044271780188306018] // 1 byte
}
```

#### Parse random base62 string
```go
package main

import (
	"fmt"
	"github.com/aohorodnyk/uid"
	"log"
)

func main() {
	// Can be used as alternative:
	// * uid.NewProvider62()
	// * uid.NewProvider62Size(8)
	// * uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBase62())
	// * uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBaseX(62))
	p := uid.NewProviderCustom(8, uid.NewRand(), uid.NewEncoderBase62())
	g, err := p.Parse("8twXZ4Nkui7")
	if err != nil {
		log.Panicln("Cannot parse random string")
	}

	fmt.Println(g.String()) // 8twXZ4Nkui7
	fmt.Println(g.Byte()) // [98 186 154 157 247 107 100 139] // 8 bytes
	fmt.Println(g.Int16()) // [-17822 -25190 27639 -29852] // 4 bytes
	fmt.Println(g.Uint16()) // [47714 40346 27639 35684] // 4 bytes
	fmt.Println(g.Int32()) // [-1650804126 -1956353033] // 2 bytes
	fmt.Println(g.Uint32()) // [2644163170 2338614263] // 2 bytes
	fmt.Println(g.Int64()) // [-8402472293521245598] // 1 byte
	fmt.Println(g.Uint64()) // [10044271780188306018] // 1 byte
}
```

## Contributing
All contributions have to follow the [CONTRIBUTING.md document](https://github.com/aohorodnyk/uid/blob/main/CONTRIBUTING.md)
If you have any questions/issues/feature requests do not hesitate to create a ticket.
