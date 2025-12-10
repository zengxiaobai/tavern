package iobuf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var _ io.ReadCloser = (*savepartReader)(nil)

type (
	EventSuccess func(buf []byte, bitIdx uint32, pos uint64, eof bool) error
	EventError   func(err error)
	EventClose   func(eof bool)
)

type savepartReader struct {
	R io.ReadCloser

	skip      bool
	eof       bool
	pos       uint64
	blockSize uint64
	buf       *bytes.Buffer

	// events
	onSuccess EventSuccess
	onError   EventError
	onClose   EventClose
}

// Read implements io.ReadCloser.
func (s *savepartReader) Read(p []byte) (n int, err error) {
	n, err = s.R.Read(p)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// flush buffer on EOF
			if err = s.flush(p, n, true); err != nil {
				s.onError(err)
			}
			return n, io.EOF
		}

		// trigger event error
		s.onError(err)
		return 0, err
	}

	if err = s.flush(p, n, false); err != nil {
		s.onError(err)
		return
	}
	return
}

// Close implements io.ReadCloser.
func (s *savepartReader) Close() error {
	if s.onClose != nil {
		s.onClose(s.eof)
	}
	return s.R.Close()
}

func (s *savepartReader) flush(data []byte, realLen int, eof bool) error {
	datalen := uint64(realLen)
	datapos := uint64(0)
	remaining := datalen

	for remaining > 0 {
		// calculate from and to within the block
		from := s.pos % s.blockSize
		to := s.blockSize - from

		// check if we need to skip
		// if we are at the beginning of a block and need to skip
		// we can skip the entire block
		if s.skip {
			// if we are at the beginning of 0 block and need to skip
			if from != 0 {
				skip := min(to, remaining)
				datapos += skip
				remaining = datalen - datapos
				s.pos += skip
				continue
			}
			s.skip = false
			continue
		}

		// full block writenow
		if uint64(s.buf.Len()) == s.blockSize {
			if err := s.writeBlock(eof); err != nil {
				return err
			}
		}

		tow := min(to, remaining)
		oldBufLen := uint64(s.buf.Len())

		s.buf.Write(data[datapos : datapos+tow])
		if oldBufLen+tow != uint64(s.buf.Len()) {
			return fmt.Errorf("partial copy - expected buffer len to be %d but it is %d",
				oldBufLen+tow, s.buf.Len())
		}

		datapos += tow
		s.pos += tow
		remaining = datalen - datapos
	}

	if eof && remaining == 0 {
		if err := s.writeBlock(eof); err != nil {
			return err
		}
	}

	return nil
}

func (s *savepartReader) writeBlock(eof bool) error {
	s.eof = eof

	buflen := uint64(s.buf.Len())
	if buflen <= 0 {
		return nil
	}

	// calc bitmap pos
	mod := uint32((s.pos - buflen) / s.blockSize)

	// trigger event success
	if err := s.onSuccess(s.buf.Bytes(), mod, s.pos, eof); err != nil {
		return fmt.Errorf("savepart_onSuccess err %w", err)
	}

	s.buf.Reset()
	return nil
}

func SavepartReader(r io.ReadCloser, blockSize int, startAt int,
	flushBuffer EventSuccess, flushFailed EventError, cleanup EventClose) io.ReadCloser {
	skip := false
	if startAt > 0 {
		skip = true
	}
	return &savepartReader{
		R: r,

		skip:      skip,
		pos:       uint64(startAt),
		blockSize: uint64(blockSize),
		onSuccess: flushBuffer,
		onError:   flushFailed,
		onClose:   cleanup,
		buf:       bytes.NewBuffer(make([]byte, 0, blockSize)),
	}
}
