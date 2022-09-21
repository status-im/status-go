# g711
--
    import "github.com/zaf/g711"

Package g711 implements encoding and decoding of G711 PCM sound data. G.711 is
an ITU-T standard for audio companding.

For usage details please see the code snippets in the cmd folder.

## Usage

```go
const (
	// Input and output formats
	Alaw = iota // Alaw G711 encoded PCM data
	Ulaw        // Ulaw G711  encoded PCM data
	Lpcm        // Lpcm 16bit signed linear data
)
```

#### func  Alaw2Ulaw

```go
func Alaw2Ulaw(alaw []byte) []byte
```
Alaw2Ulaw performs direct A-law to u-law data conversion

#### func  Alaw2UlawFrame

```go
func Alaw2UlawFrame(frame uint8) uint8
```
Alaw2UlawFrame directly converts an A-law frame to u-law

#### func  DecodeAlaw

```go
func DecodeAlaw(pcm []byte) []byte
```
DecodeAlaw decodes A-law PCM data to 16bit LPCM

#### func  DecodeAlawFrame

```go
func DecodeAlawFrame(frame uint8) int16
```
DecodeAlawFrame decodes an A-law PCM frame to 16bit LPCM

#### func  DecodeUlaw

```go
func DecodeUlaw(pcm []byte) []byte
```
DecodeUlaw decodes u-law PCM data to 16bit LPCM

#### func  DecodeUlawFrame

```go
func DecodeUlawFrame(frame uint8) int16
```
DecodeUlawFrame decodes a u-law PCM frame to 16bit LPCM

#### func  EncodeAlaw

```go
func EncodeAlaw(lpcm []byte) []byte
```
EncodeAlaw encodes 16bit LPCM data to G711 A-law PCM

#### func  EncodeAlawFrame

```go
func EncodeAlawFrame(frame int16) uint8
```
EncodeAlawFrame encodes a 16bit LPCM frame to G711 A-law PCM

#### func  EncodeUlaw

```go
func EncodeUlaw(lpcm []byte) []byte
```
EncodeUlaw encodes 16bit LPCM data to G711 u-law PCM

#### func  EncodeUlawFrame

```go
func EncodeUlawFrame(frame int16) uint8
```
EncodeUlawFrame encodes a 16bit LPCM frame to G711 u-law PCM

#### func  Ulaw2Alaw

```go
func Ulaw2Alaw(ulaw []byte) []byte
```
Ulaw2Alaw performs direct u-law to A-law data conversion

#### func  Ulaw2AlawFrame

```go
func Ulaw2AlawFrame(frame uint8) uint8
```
Ulaw2AlawFrame directly converts a u-law frame to A-law

#### type Decoder

```go
type Decoder struct {
}
```

Decoder reads G711 PCM data and decodes it to 16bit 8000Hz LPCM

#### func  NewAlawDecoder

```go
func NewAlawDecoder(reader io.Reader) (*Decoder, error)
```
NewAlawDecoder returns a pointer to a Decoder that implements an io.Reader. It
takes as input the source data Reader.

#### func  NewUlawDecoder

```go
func NewUlawDecoder(reader io.Reader) (*Decoder, error)
```
NewUlawDecoder returns a pointer to a Decoder that implements an io.Reader. It
takes as input the source data Reader.

#### func (*Decoder) Read

```go
func (r *Decoder) Read(p []byte) (i int, err error)
```
Read decodes G711 data. Reads up to len(p) bytes into p, returns the number of
bytes read and any error encountered.

#### func (*Decoder) Reset

```go
func (r *Decoder) Reset(reader io.Reader) error
```
Reset discards the Decoder state. This permits reusing a Decoder rather than
allocating a new one.

#### type Encoder

```go
type Encoder struct {
}
```

Encoder encodes 16bit 8000Hz LPCM data to G711 PCM or directly transcodes
between A-law and u-law

#### func  NewAlawEncoder

```go
func NewAlawEncoder(writer io.Writer, input int) (*Encoder, error)
```
NewAlawEncoder returns a pointer to an Encoder that implements an io.Writer. It
takes as input the destination data Writer and the input encoding format.

#### func  NewUlawEncoder

```go
func NewUlawEncoder(writer io.Writer, input int) (*Encoder, error)
```
NewUlawEncoder returns a pointer to an Encoder that implements an io.Writer. It
takes as input the destination data Writer and the input encoding format.

#### func (*Encoder) Reset

```go
func (w *Encoder) Reset(writer io.Writer) error
```
Reset discards the Encoder state. This permits reusing an Encoder rather than
allocating a new one.

#### func (*Encoder) Write

```go
func (w *Encoder) Write(p []byte) (i int, err error)
```
Write encodes G711 Data. Writes len(p) bytes from p to the underlying data
stream, returns the number of bytes written from p (0 <= n <= len(p)) and any
error encountered that caused the write to stop early.
