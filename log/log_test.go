package log_test

import (
	"testing"

	"github.com/1garo/kival/log"
	"github.com/stretchr/testify/assert"
)

func TestReadAt(t *testing.T) {
	active, _, _, err := log.Open("test-data")

	key := []byte("key")
	val := []byte("value")
	pos, err := active.Append(key, val)
	assert.NoError(t, err)

	data, err := active.ReadAt(pos)
	assert.NoError(t, err)
	assert.Equal(t, val, data)
}

// Append(key, val []byte) (pos LogPosition, err error)
// ReadAt(pos LogPosition) ([]byte, error)
// Size() int64
// ID() uint32
// Close() error
// MakeReadOnly()
