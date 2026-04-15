//go:build real

package claudebox

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func realClient(t *testing.T) *Client {
	t.Helper()

	_ = godotenv.Load(".env.test")

	url := os.Getenv("CLAUDEBOX_URL")
	token := os.Getenv("CLAUDEBOX_TOKEN")

	require.NotEmpty(t, url,
		"CLAUDEBOX_URL must be set in .env.test or environment",
	)
	require.NotEmpty(t, token,
		"CLAUDEBOX_TOKEN must be set in .env.test or environment",
	)

	return New(url, WithToken(token))
}
