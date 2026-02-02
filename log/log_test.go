package log_test

import (
	"testing"

	"github.com/1garo/kival/log"
	"github.com/1garo/kival/record"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLog(t *testing.T) log.Log {
	t.Helper()

	dir := t.TempDir()
	l, err := log.New(1, dir)
	require.NoError(t, err)

	t.Cleanup(func() { _ = l.Close() })
	return l
}

func TestLog_ReadAt_ReturnsAppendedValue(t *testing.T) {
	activeLog := newTestLog(t)

	key := []byte("key")
	val := []byte("value")
	pos, err := activeLog.Append(key, val)
	require.NoError(t, err)

	data, err := activeLog.ReadAt(pos)
	assert.NoError(t, err)
	assert.Equal(t, val, data, "val should be the same as the read from data")
}

func TestLog_ReadAt_MultipleRecords(t *testing.T) {
	activeLog := newTestLog(t)

	val1 := []byte("v1")
	p1, err := activeLog.Append([]byte("k1"), val1)
	require.NoError(t, err)
	val2 := []byte("v2")
	p2, err := activeLog.Append([]byte("k2"), val2)
	require.NoError(t, err)
	val3 := []byte("v3")
	p3, err := activeLog.Append([]byte("k3"), val3)
	require.NoError(t, err)

	v2, err := activeLog.ReadAt(p2)
	assert.NoError(t, err)
	assert.Equal(t, val2, v2)

	v1, err := activeLog.ReadAt(p1)
	assert.NoError(t, err)
	assert.Equal(t, val1, v1)

	v3, err := activeLog.ReadAt(p3)
	assert.NoError(t, err)
	assert.Equal(t, val3, v3)
}

func TestLog_ReadAt_InvalidPositionReturnsError(t *testing.T) {
	activeLog := newTestLog(t)

	p := log.LogPosition{ValuePos: 0}
	b, err := activeLog.ReadAt(p)
	assert.ErrorIs(t, err, record.ErrPartialWrite, "should fail because position is invalid")
	assert.Equal(t, 0, len(b), "should return empty data")
}

func TestLog_ReadAt_AfterCloseReturnsError(t *testing.T) {
	activeLog := newTestLog(t)

	p, err := activeLog.Append([]byte("k1"), []byte("v1"))
	require.NoError(t, err)
	err = activeLog.Close()
	require.NoError(t, err)

	b, err := activeLog.ReadAt(p)
	assert.ErrorIs(t, err, log.ErrLogClosed, "should fail because log is closed")
	assert.Equal(t, 0, len(b), "should return empty data")
}

func TestLog_ReadAt_TruncatedRecordReturnsError(t *testing.T) {
	activeLog := newTestLog(t)

	p, err := activeLog.Append([]byte("k1"), []byte("v1"))
	require.NoError(t, err)

	p.ValuePos += 1
	b, err := activeLog.ReadAt(p)
	assert.ErrorIs(t, err, record.ErrPartialWrite, "should fail because log is closed")
	assert.Equal(t, 0, len(b), "should return empty data")
}

func TestLog_Append_InsertRecord(t *testing.T) {
	activeLog := newTestLog(t)

	val := []byte("v1")
	p, err := activeLog.Append([]byte("k1"), val)
	require.NoError(t, err)

	assert.Equal(t, p.FileID, activeLog.ID(), "should return correct file ID")
	assert.Equal(t, p.ValuePos, int64(0), "should return correct position")
	assert.Equal(t, p.ValueSize, uint32(len(val)), "should return correct value size")
}

func TestLog_Append_ReadOnlySegmentError(t *testing.T) {
	activeLog := newTestLog(t)

	activeLog.MarkReadOnly()

	val := []byte("v1")
	p, err := activeLog.Append([]byte("k1"), val)
	assert.ErrorIs(t, err, log.ErrReadOnlySegment, "should fail because log is read-only")
	assert.True(t, p == log.LogPosition{}, "position should be empty")
}
