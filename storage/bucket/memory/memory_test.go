package memory

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/stretchr/testify/assert"
)

func TestMemFs(t *testing.T) {
	fs := vfs.NewMem()
	err := fs.MkdirAll("/memory/f/2c/", 0o700)
	assert.NoError(t, err)

	fp := filepath.Join("/memory/f/2c/aaaa")

	file, err := fs.OpenReadWrite(fp, vfs.WriteCategoryUnspecified)
	assert.NoError(t, err)

	n, err := file.Write([]byte("hello, world"))
	assert.NoError(t, err)
	assert.Equal(t, n, 12)

	assert.NoError(t, file.Close())

	readFs, err := fs.Open(fp)
	assert.NoError(t, err)

	written, err := io.Copy(os.Stdout, readFs)
	assert.NoError(t, err)
	assert.Equal(t, written, int64(12))
	assert.NoError(t, readFs.Close())

	t.Log()
	t.Log(fs.String())
}
