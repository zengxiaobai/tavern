package iobuf

import (
	"io"
	"net/http"
)

// ProxyCallback defines a function type that returns an HTTP response and an error when called.
type ProxyCallback func() (*http.Response, error)

// asyncReader is a struct that wraps an io.ReadCloser for reading data asynchronously.
// It captures any errors encountered during reading for later retrieval.
type asyncReader struct {
	R   io.ReadCloser
	err error
}

// AsyncReadCloser creates an asynchronous io.ReadCloser that invokes a ProxyCallback to process data in the background.
func AsyncReadCloser(proxy ProxyCallback) io.ReadCloser {
	pr, pw := io.Pipe()

	ar := &asyncReader{R: pr}
	go func() {
		resp, err := proxy()
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			_ = pw.Close()
		}()

		if err != nil {
			ar.err = err
			_ = pw.CloseWithError(err)
			return
		}

		if resp == nil || resp.Body == nil {
			// ar.err = io.ErrUnexpectedEOF
			_ = pw.CloseWithError(io.ErrUnexpectedEOF)
			return
		}

		_, err1 := io.Copy(pw, resp.Body)
		if err1 != nil {
			ar.err = err1
			_ = pw.CloseWithError(err1)
		}
	}()

	return ar
}

// Read reads data into the provided byte slice and returns the number of bytes read and an error, if any occurred.
func (r *asyncReader) Read(p []byte) (n int, err error) {
	n, err = r.R.Read(p)
	if err == io.EOF {
		return n, err
	}
	if err != nil {
		return n, err
	}
	if r.err != nil {
		return n, r.err
	}
	return n, nil
}

// Close releases the underlying resource if it is non-nil and returns any error encountered during the operation.
func (r *asyncReader) Close() error {
	if r.R != nil {
		return r.R.Close()
	}
	return nil
}
