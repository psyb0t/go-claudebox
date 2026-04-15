//go:build real

package claudebox

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealListFiles(t *testing.T) {
	c := realClient(t)

	resp, err := c.ListFiles(
		context.Background(), "",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Path)
	assert.NotNil(t, resp.Entries)
}

func TestRealWriteReadDeleteFile(t *testing.T) {
	c := realClient(t)
	ctx := context.Background()

	name := fmt.Sprintf(
		"go-claudebox-test-%d.txt",
		time.Now().UnixNano(),
	)
	content := []byte("integration test content")

	// write
	wr, err := c.WriteFile(ctx, name, content)
	require.NoError(t, err)
	assert.Equal(t, "ok", wr.Status)
	assert.Equal(t, len(content), wr.Size)

	// read back
	rr, err := c.ReadFile(ctx, name)
	require.NoError(t, err)

	defer func() { _ = rr.Body.Close() }()

	assert.NotEmpty(t, rr.ContentType)
	assert.Greater(t, rr.ContentLength, int64(0))

	data, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	// verify in listing
	lr, err := c.ListFiles(ctx, "")
	require.NoError(t, err)

	found := false

	for _, e := range lr.Entries {
		if e.Name == name {
			found = true
			assert.Equal(t, "file", e.Type)
			assert.Equal(t, int64(len(content)), e.Size)
		}
	}

	assert.True(t, found,
		"uploaded file %q not found in listing", name,
	)

	// delete
	dr, err := c.DeleteFile(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, "ok", dr.Status)

	// verify gone
	_, err = c.ReadFile(ctx, name)
	require.Error(t, err)
}

func TestRealReadFileBinary(t *testing.T) {
	c := realClient(t)
	ctx := context.Background()

	name := fmt.Sprintf(
		"go-claudebox-bin-%d.dat",
		time.Now().UnixNano(),
	)
	binary := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	_, err := c.WriteFile(ctx, name, binary)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = c.DeleteFile(ctx, name)
	})

	rr, err := c.ReadFile(ctx, name)
	require.NoError(t, err)

	defer func() { _ = rr.Body.Close() }()

	data, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	assert.Equal(t, binary, data)
}
