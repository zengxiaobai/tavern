package mod

import (
	"bytes"
	"strings"
)

const defaultBufferSize = 1 << 8

type FieldBuffer struct {
	data bytes.Buffer
	sep  byte
}

func NewFieldBuffer(sep byte) *FieldBuffer {
	var b bytes.Buffer
	b.Grow(defaultBufferSize)

	return &FieldBuffer{
		data: b,
		sep:  sep,
	}
}

// Append normal string append to buffer
func (b *FieldBuffer) Append(s string) {
	b.append(s, false)
}

// FAppend replace space to + and append to buffer
func (b *FieldBuffer) FAppend(s string) {
	b.append(s, true)
}

// Bytes return a slice of length b.Len() holding the unread portion of the buffer.
func (b *FieldBuffer) Bytes() []byte {
	return b.data.Bytes()
}

// String returns the contents of buffer.
func (b *FieldBuffer) String() string {
	return b.data.String()
}

func (b *FieldBuffer) append(s string, rep bool) {
	s = emptyWrap(s)
	if rep {
		s = strings.ReplaceAll(s, " ", "+")
	}
	if b.data.Len() > 0 {
		b.data.WriteByte(b.sep)
	}
	b.data.WriteString(s)
}

func emptyWrap(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
