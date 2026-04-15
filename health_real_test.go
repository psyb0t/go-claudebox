//go:build real

package claudebox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealHealth(t *testing.T) {
	c := realClient(t)

	resp, err := c.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestRealStatus(t *testing.T) {
	c := realClient(t)

	resp, err := c.Status(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp.BusyWorkspaces)
}
