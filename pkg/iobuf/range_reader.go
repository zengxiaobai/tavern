package iobuf

import "io"

// rangeReader is a type that implements io.ReadCloser and provides reading within specified byte range boundaries.
type rangeReader struct {
	R        io.ReadCloser
	newStart int
	newEnd   int
	rawStart int
	rawEnd   int
	offset   int
}

// RangeReader returns a ReadCloser that reads a specified byte range from an underlying ReadCloser.
func RangeReader(r io.ReadCloser, newStart int, newEnd int, rawStart int, rawEnd int) io.ReadCloser {
	return &rangeReader{
		R:        r,
		newStart: newStart,
		newEnd:   newEnd,
		rawStart: rawStart,
		rawEnd:   rawEnd,
		offset:   newStart,
	}
}

// Read reads up to len(p) bytes into p from the underlying stream within the configured range boundaries.
// Skips data before the rawStart offset and discards any data outside the specified range.
// Returns the number of bytes read and an error, if any.
func (r *rangeReader) Read(p []byte) (int, error) {
	// skip to rawStart
	if r.offset < r.rawStart {
		skipN, err := io.CopyN(io.Discard, r.R, int64(r.rawStart-r.offset))
		if err != nil {
			return 0, err
		}
		r.offset += int(skipN)
	}

	// 一直读取用户的数据
	n, err := r.R.Read(p)

	// 进行判断是否已经读完用户的数据
	// 并进行剩余数据的 Discard 动作
	cur := r.offset + n
	if cur > r.rawEnd {
		// 本次读取范围 用户的真实数据长度
		remaining := r.rawEnd - r.offset + 1
		// 距离读取完毕还剩余的长度
		discardSize := min(r.newEnd, r.newEnd-cur+1)
		// 如果还有剩余的数据, 则进行 Discard
		if discardSize > 0 {
			skipN, _ := io.CopyN(io.Discard, r.R, int64(discardSize))
			r.offset += int(skipN)
		} else {
			n += discardSize
		}
		r.offset += n
		// 结束提前跳出
		return remaining, io.EOF
	}

	r.offset += n
	return n, err
}

// Close closes the underlying io.ReadCloser and releases any resources associated with it.
func (r *rangeReader) Close() error {
	return r.R.Close()
}
