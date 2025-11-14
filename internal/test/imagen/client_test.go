package imagen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"instant-hdr-backend/internal/imagen"
)

func TestClient_RetryWithBackoff(t *testing.T) {
	client := imagen.NewClient("https://api.test.com/v1/", "test-key")

	callCount := 0
	err := client.RetryWithBackoff(func() error {
		callCount++
		if callCount < 3 {
			return assert.AnError
		}
		return nil
	}, 3)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestClient_RetryWithBackoff_Exhausted(t *testing.T) {
	client := imagen.NewClient("https://api.test.com/v1/", "test-key")

	err := client.RetryWithBackoff(func() error {
		return assert.AnError
	}, 3)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 retries")
}

