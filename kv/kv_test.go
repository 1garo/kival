package kv_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/1garo/kival/kv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKV(t *testing.T, dir string) kv.KV {
	t.Helper()
	db, err := kv.New(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		if db != nil {
			_ = os.RemoveAll(dir)
		}
	})
	return db
}

func listDataFiles(dir string) []string {
	var files []string
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".data" {
			files = append(files, e.Name())
		}
	}
	return files
}

func forceRotation(db kv.KV, count int) {
	val := []byte("this is a long value that will fill the log") // ~43 bytes, ~7-8 per file
	for i := 0; i < count; i++ {
		db.Put([]byte("key"+string(rune('a'+i%26))), val)
	}
}

func TestKV_Merge_CreatesCompactedLog(t *testing.T) {
	dir := t.TempDir()
	db := newTestKV(t, dir)

	err := db.Put([]byte("key1"), []byte("value1"))
	require.NoError(t, err)

	filesBefore := listDataFiles(dir)

	err = db.Merge()
	require.NoError(t, err)

	filesAfter := listDataFiles(dir)
	assert.NotEmpty(t, filesAfter, "Merge should create at least one data file")
	assert.LessOrEqual(t, len(filesAfter), len(filesBefore), "Merge should reduce number of files")
}

func TestKV_Merge_DeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	db := newTestKV(t, dir)

	forceRotation(db, 60) // 60 keys × ~59 bytes (43+16 header) = ~3540 bytes = ~2 files

	filesBefore := listDataFiles(dir)
	if len(filesBefore) <= 1 {
		t.Skip("needs multiple log files")
	}

	db.Put([]byte("key1"), []byte("val")) // small value for merge

	err := db.Merge()
	require.NoError(t, err)

	filesAfter := listDataFiles(dir)
	assert.Less(t, len(filesAfter), len(filesBefore), "old files should be deleted")
}

func TestKV_Merge_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	db := newTestKV(t, dir)

	err := db.Merge()
	require.NoError(t, err)
}

func TestKV_Merge_UpdatesKeyPositions(t *testing.T) {
	dir := t.TempDir()
	db := newTestKV(t, dir)

	forceRotation(db, 10) // creates multiple files

	err := db.Put([]byte("key1"), []byte("val")) // small value for merge
	require.NoError(t, err)

	err = db.Merge()
	require.NoError(t, err)

	val, err := db.Get([]byte("key1"))
	require.NoError(t, err)
	assert.Equal(t, "val", string(val))

	files := listDataFiles(dir)
	assert.Equal(t, 1, len(files), "should have only compacted log")
}
