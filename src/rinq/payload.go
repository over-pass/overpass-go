package rinq

import (
	"bytes"
	"reflect"
	"sync"

	"github.com/rinq/rinq-go/src/internal/x/bufferpool"
	"github.com/rinq/rinq-go/src/internal/x/cbor"
	"github.com/ugorji/go/codec"
)

// Payload is an immutable, application-defined value that is included in a
// command request, command response, or inter-session notification.
//
// A nil-payload pointer is equivalent to a payload with a value of nil.
//
// Payloads must be closed by the application when no longer required. This
// includes payloads constructed by calling NewPayload() or NewPayloadFromBytes(),
// as well as any payload returned by a Rinq operation (such as Session.Call()),
// or passed to a callback function that was provided by the application.
//
// Payloads are NOT safe for concurrent use. To share a payload across multiple
// goroutines, call Payload.Clone() to obtain a second payload that references
// the same underlying data.
//
// Payload values can be any value that can be represented using CBOR encoding.
// See http://cbor.io/ for more information.
//
// Payloads are modeled in this way to allow an application to forward incoming
// payloads without the need to decode and re-encode them.
type Payload struct {
	data *payloadData
}

// NewPayload creates a new payload from an arbitrary value.
func NewPayload(v interface{}) *Payload {
	if v == nil {
		return nil
	}

	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr && r.IsNil() {
		return nil
	}

	return &Payload{
		&payloadData{
			value:    v,
			hasValue: true,
			refCount: 1,
		},
	}
}

// NewPayloadFromBytes creates a new payload from a binary representation.
// Ownership of the byte-slice is transferred to the payload. An empty
// byte-slice is equivalent to the nil value.
func NewPayloadFromBytes(buf []byte) *Payload {
	if len(buf) == 0 {
		return nil
	}

	return &Payload{
		&payloadData{
			buffer:   bytes.NewBuffer(buf),
			refCount: 1,
		},
	}
}

// Clone returns a copy of this payload.
func (p *Payload) Clone() *Payload {
	if p == nil || p.data == nil {
		return nil
	}

	p.data.writeMutex.Lock()
	defer p.data.writeMutex.Unlock()

	p.data.refCount++

	return &Payload{p.data}
}

// Bytes returns the binary representation of the payload, in CBOR encoding.
//
// The returned byte-slice is invalidated when the payload is closed, it must be
// copied if it is intended to be used for longer than the lifetime of the
// payload.
//
// If the payload was created from a non-empty byte-slice, the return value is
// always that same byte-slice, unless the payload has been closed.
//
// If the payload was created from a nil value, the returned byte-slice is nil.
func (p *Payload) Bytes() []byte {
	if p == nil || p.data == nil {
		return nil
	}

	p.data.readMutex.Lock()
	defer p.data.readMutex.Unlock()

	if p.data.buffer != nil {
		return p.data.buffer.Bytes()
	}

	p.data.writeMutex.Lock()
	defer p.data.writeMutex.Unlock()

	buffer := bufferpool.Get()
	cbor.MustEncode(buffer, p.data.value)
	p.data.buffer = buffer

	return buffer.Bytes()
}

// Len returns the encoded payload length, in bytes.
// A length of zero indicates a nil payload value.
func (p *Payload) Len() int {
	return len(p.Bytes())
}

// Decode unpacks the payload into the given value.
func (p *Payload) Decode(value interface{}) error {
	buf := p.Bytes()
	if buf == nil {
		buf = cbor.Nil
	}

	return cbor.DecodeBytes(buf, value)
}

// Value returns the payload value.
func (p *Payload) Value() interface{} {
	if p == nil || p.data == nil {
		return nil
	}

	p.data.readMutex.Lock()
	defer p.data.readMutex.Unlock()

	if p.data.hasValue {
		return p.data.value
	}

	p.data.writeMutex.Lock()
	defer p.data.writeMutex.Unlock()

	cbor.MustDecodeBytes(p.data.buffer.Bytes(), &p.data.value)
	p.data.hasValue = true

	return p.data.value
}

// Close releases any resources held by the payload, resetting the payload to
// represent the nil value.
func (p *Payload) Close() {
	if p == nil || p.data == nil {
		return
	}

	data := p.data
	p.data = nil

	data.writeMutex.Lock()
	defer data.writeMutex.Unlock()

	data.refCount--

	if data.refCount == 0 && data.buffer != nil {
		bufferpool.Put(data.buffer)
	}
}

// String returns a human-readable representation of the payload.
// No guarantees are made about the format of the string.
func (p *Payload) String() string {
	buffer := bufferpool.Get()
	defer bufferpool.Put(buffer)

	encoder := jsonEncoders.Get().(*codec.Encoder)
	encoder.Reset(buffer)
	encoder.MustEncode(p.Value())

	return buffer.String()
}

type payloadData struct {
	readMutex  sync.Mutex
	writeMutex sync.Mutex

	// The binary representation of the payload. If the payload has never been
	// encoded, buffer is nil.
	buffer *bytes.Buffer

	// The payload value. If the payload has never been decoded, value is nil
	// and hasValue is false.
	value interface{}

	// Indicates whether the value has been populated.
	hasValue bool

	// refCount is the number of payload structures that are pointing to this
	// element.
	refCount uint
}

var jsonHandle codec.JsonHandle
var jsonEncoders = sync.Pool{
	New: func() interface{} {
		return codec.NewEncoder(nil, &jsonHandle)
	},
}
