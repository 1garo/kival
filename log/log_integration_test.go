//go:build integration

package log_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	assert.ErrorIs(t, err, record.ErrPartialWrite, "should fail because log position is invalid")
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

func TestNew_LogFileCreation(t *testing.T) {
	dir := t.TempDir()

	l, err := log.New(1, dir)
	require.NoError(t, err)
	defer l.Close()

	expectedPath := filepath.Join(dir, "1.data")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "log file should exist")

	assert.Equal(t, uint32(1), l.ID(), "should have correct file ID")
	assert.Equal(t, int64(0), l.Size(), "new file should be empty")
}

func TestNew_DirectoryCreation(t *testing.T) {
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "nested", "log", "dir")

	l, err := log.New(42, nestedDir)
	require.NoError(t, err)
	defer l.Close()

	expectedPath := filepath.Join(nestedDir, "42.data")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "nested directories and log file should exist")

	info, err := os.Stat(nestedDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir(), "path should be a directory")
}

func TestNew_FilePermissions(t *testing.T) {
	dir := t.TempDir()

	l, err := log.New(1, dir)
	require.NoError(t, err)
	defer l.Close()

	filePath := filepath.Join(dir, "1.data")
	info, err := os.Stat(filePath)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0644), info.Mode().Perm(), "file should have 0644 permissions")
}

func TestOpen_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	active, logs, index, err := log.Open(dir)
	require.NoError(t, err)
	defer active.Close()

	assert.Equal(t, uint32(1), active.ID(), "should create log with ID 1 in empty directory")
	assert.Empty(t, logs, "should have no readonly logs")
	assert.Empty(t, index, "should have empty index")

	expectedPath := filepath.Join(dir, "1.data")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "log file should be created")
}

func TestOpen_ExistingFiles(t *testing.T) {
	dir := t.TempDir()

	createTestLogFile(t, filepath.Join(dir, "1.data"), []byte("test1"))
	createTestLogFile(t, filepath.Join(dir, "2.data"), []byte("test2"))
	createTestLogFile(t, filepath.Join(dir, "3.data"), []byte("test3"))

	active, logs, _, err := log.Open(dir)
	require.NoError(t, err)
	defer active.Close()

	// Should use highest ID as active log
	assert.Equal(t, uint32(3), active.ID(), "should use highest file ID as active")
	assert.Len(t, logs, 2, "should have 2 readonly logs")
	assert.Contains(t, logs, uint32(1), "should contain file ID 1")
	assert.Contains(t, logs, uint32(2), "should contain file ID 2")
}

func TestOpen_MultipleFilesSorted(t *testing.T) {
	dir := t.TempDir()

	// Create files in random order
	createTestLogFile(t, filepath.Join(dir, "5.data"), []byte("test5"))
	createTestLogFile(t, filepath.Join(dir, "2.data"), []byte("test2"))
	createTestLogFile(t, filepath.Join(dir, "8.data"), []byte("test8"))
	createTestLogFile(t, filepath.Join(dir, "1.data"), []byte("test1"))

	active, logs, _, err := log.Open(dir)
	require.NoError(t, err)
	defer active.Close()

	// Should use highest ID as active log
	assert.Equal(t, uint32(8), active.ID(), "should use highest file ID as active")

	// Verify logs are sorted correctly
	expectedIDs := []uint32{1, 2, 5}
	for _, id := range expectedIDs {
		assert.Contains(t, logs, id, "should contain file ID %d", id)
	}
}

func TestOpen_NonexistentDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "nonexistent")

	// Should create directory and new log file
	active, logs, index, err := log.Open(subdir)
	require.NoError(t, err)
	defer active.Close()

	assert.Equal(t, uint32(1), active.ID(), "should create new log with ID 1")
	assert.Empty(t, logs, "should have no readonly logs")
	assert.Empty(t, index, "should have empty index")
}

func TestParseFileID_ValidNames(t *testing.T) {
	// Helper function to test file ID parsing (mirrors internal parseFileID)
	testParseFileID := func(name string) uint32 {
		base := filepath.Base(name)
		idStr := strings.TrimSuffix(base, ".data")
		id, _ := strconv.ParseUint(idStr, 10, 32)
		return uint32(id)
	}

	testCases := []struct {
		name     string
		expected uint32
	}{
		{"1.data", 1},
		{"42.data", 42},
		{"999.data", 999},
		{"0.data", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := testParseFileID(tc.name)
			assert.Equal(t, tc.expected, result, "should parse file ID correctly")
		})
	}
}

// Helper function to create test log files
func createTestLogFile(t *testing.T, path string, content []byte) {
	t.Helper()

	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)
}

func TestAppend_CapacityExceeded(t *testing.T) {
	l := newTestLog(t)

	// Create a large key-value pair that will consume most of MaxDataFileSize (500 bytes)
	// Record size = header(16) + keySize(100) + valueSize(380) = 496 bytes
	largeKey := make([]byte, 100)
	largeValue := make([]byte, 380)

	// First append should succeed
	_, err := l.Append(largeKey, largeValue)
	require.NoError(t, err)

	t.Logf("After first append, log size: %d", l.Size())

	// Second append should fail due to capacity exceeded (needs 16+5+5=26 more bytes)
	smallKey := []byte("small")
	smallValue := []byte("value")
	_, err = l.Append(smallKey, smallValue)
	assert.ErrorIs(t, err, log.ErrCapacityExceeded, "should fail when capacity is exceeded")
}

func TestAppend_ExactlyAtCapacity(t *testing.T) {
	l := newTestLog(t)

	// Create a key-value pair that exactly fits remaining capacity
	// Start with a small record to reduce capacity
	smallKey := []byte("small")
	smallValue := []byte("value")
	_, err := l.Append(smallKey, smallValue)
	require.NoError(t, err)

	// Calculate remaining capacity and create exact fit record
	// MaxDataFileSize is 500, header is 16 bytes, current size includes first record
	remainingCapacity := log.MaxDataFileSize - int(l.Size())
	keySize := 8
	valueSize := remainingCapacity - int(record.HeaderSize) - keySize // 16 for header
	t.Logf("Remaining capacity: %d", remainingCapacity)

	assert.Greater(t, valueSize, 0, "Not enough remaining capacity for exact capacity test")

	exactKey := make([]byte, keySize)
	exactValue := make([]byte, valueSize)

	pos, err := l.Append(exactKey, exactValue)
	assert.NoError(t, err, "should append exactly at capacity boundary")
	assert.Equal(t, uint32(valueSize), pos.ValueSize, "should record correct value size")

	_, err = l.Append([]byte("fail"), []byte("test"))
	assert.ErrorIs(t, err, log.ErrCapacityExceeded, "should fail after reaching capacity")
}

func TestAppend_MultipleRecordsUntilFull(t *testing.T) {
	l := newTestLog(t)

	var recordCount int
	for i := 0; ; i++ {
		key := fmt.Appendf([]byte{}, "key%d", i)
		value := fmt.Appendf([]byte{}, "value%d", i)

		_, err := l.Append(key, value)
		if err != nil {
			assert.ErrorIs(t, err, log.ErrCapacityExceeded, "should fail with capacity exceeded")
			break
		}
		recordCount++
	}

	assert.Greater(t, recordCount, 0, "should have multiple records")
	assert.Less(t, l.Size(), int64(500), "should not exceed max capacity")
}

func TestMarkReadOnly_PreventsAppend(t *testing.T) {
	l := newTestLog(t)
	l.MarkReadOnly()

	pos, err := l.Append([]byte("readonly"), []byte("test'"))

	assert.ErrorIs(t, err, log.ErrReadOnlySegment, "should fail trying to write to a read-only log")
	assert.Empty(t, pos, "should return empty position on error")
}

func TestMarkReadOnly_AllowsReads(t *testing.T) {
	l := newTestLog(t)

	// First append data normally
	value := []byte("testvalue")
	pos, err := l.Append([]byte("testkey"), value)
	require.NoError(t, err)

	l.MarkReadOnly()

	data, err := l.ReadAt(pos)
	assert.NoError(t, err, "should allow reads after marking log read-only")
	assert.Equal(t, value, data, "should return correct data")
}
