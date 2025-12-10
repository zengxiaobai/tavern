package iobuf_test

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
)

func markbuf(size int64) []byte {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return buf
}

func checksum(R io.ReadCloser, start, end int64) string {
	hash := md5.New()

	n, _ := io.CopyN(io.Discard, R, start)
	log.Printf("head skipAt: %d\n", n)
	n, err := io.CopyN(hash, R, end-start+1)
	log.Printf("body readAt: %d\n", n)
	if err != nil {
		return ""
	}
	n, _ = io.Copy(io.Discard, R)
	log.Printf("tail skipAt: %d", n)
	return hex.EncodeToString(hash.Sum(nil))
}
