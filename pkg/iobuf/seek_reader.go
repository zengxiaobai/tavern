package iobuf

import (
	"fmt"
	"io"
	"sync"
)

// seekReadCloser is a wrapper around io.ReadSeekCloser that ensures a specific offset is set before any read operations.
type seekReadCloser struct {
	R      io.ReadSeekCloser
	offset int64
	s      sync.Once
}

// SeekReadCloser creates an io.ReadCloser that begins reading from the specified offset in the provided io.ReadSeekCloser.
func SeekReadCloser(R io.ReadSeekCloser, offset int64) io.ReadCloser {
	return &seekReadCloser{
		R:      R,
		offset: offset,
		s:      sync.Once{},
	}
}

// Close closes the underlying ReadSeekCloser resource and releases any associated resources.
func (s *seekReadCloser) Close() error {
	return s.R.Close()
}

// Read reads data into p from the underlying io.ReadSeekCloser, ensuring the initial offset is applied exactly once.
func (s *seekReadCloser) Read(p []byte) (n int, err error) {
	s.s.Do(func() {
		skip, err := s.R.Seek(s.offset, io.SeekStart)
		if err != nil {
			panic(err)
		}
		if skip != s.offset {
			panic(fmt.Sprintf("seek failed, got %d, want %d", skip, s.offset))
		}
	})
	n, err = s.R.Read(p)
	return
}

// WriteTo writes data from the underlying io.ReadSeekCloser to the provided io.Writer starting from the specified offset.
func (s *seekReadCloser) WriteTo(w io.Writer) (n int64, err error) {
	s.s.Do(func() {
		skip, err := s.R.Seek(s.offset, io.SeekStart)
		if err != nil {
			panic(err)
		}
		if skip != s.offset {
			panic(fmt.Sprintf("seek failed, got %d, want %d", skip, s.offset))
		}
	})
	return io.Copy(w, s.R)
}
