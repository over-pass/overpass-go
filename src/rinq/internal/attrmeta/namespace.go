package attrmeta

import (
	"bytes"

	"github.com/rinq/rinq-go/src/rinq/internal/x/bufferpool"
)

// Namespace maps attribute keys to AttrRevs
type Namespace map[string]Attr

// Clone returns a copy of the attribute table.
func (ns Namespace) Clone() Namespace {
	r := Namespace{}

	for k, v := range ns {
		r[k] = v
	}

	return r
}

// WriteTo writes a respresentation of t to buf.
// Non-frozen attributes with empty-values are omitted.
func (ns Namespace) WriteTo(buf *bytes.Buffer) (notEmpty bool) {
	buf.WriteRune('{')
	notEmpty = ns.writeTo(buf)
	buf.WriteRune('}')
	return
}

// WriteWithNameTo writes a respresentation of t to buf, including a
// namespace name. Non-frozen attributes with empty-values are omitted.
func (ns Namespace) WriteWithNameTo(buf *bytes.Buffer, name string) (notEmpty bool) {
	buf.WriteString(name)
	buf.WriteString("::")
	buf.WriteRune('{')
	notEmpty = ns.writeTo(buf)
	buf.WriteRune('}')
	return
}

func (ns Namespace) writeTo(buf *bytes.Buffer) (notEmpty bool) {
	for _, attr := range ns {
		if !attr.IsFrozen && attr.Value == "" {
			continue
		}

		if notEmpty {
			buf.WriteString(", ")
		} else {
			notEmpty = true
		}

		attr.WriteTo(buf)
	}

	return
}

func (ns Namespace) String() string {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	ns.WriteTo(buf)

	return buf.String()
}
