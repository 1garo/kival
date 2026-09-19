package main

import (
	"errors"
	l "log"

	"github.com/1garo/kival/kv"
)

func main() {
	db, err := kv.New(kv.DefaultDBPath)
	if err != nil {
		l.Fatalf("failed to open the db: %v", err)
	}
	defer db.Close()

	key := []byte("bar")
	val := []byte("baz")
	if err := db.Put(key, val); err != nil {
		l.Fatalf("failed to set the key=%s: %v", key, err)
	}

	l.Printf("new key added to the db: %s\n", key)

	data, err := db.Get(key)
	if err != nil {
		l.Fatalf("failed to get the key=%s: %v", key, err)
	}
	l.Printf("data retrieved for %s: %s\n", key, data)

	if err := db.Delete(key); err != nil {
		l.Fatalf("failed to delete the key=%s: %v", key, err)
	}
	l.Println("successfully delete the key")

	if _, err := db.Get(key); !errors.Is(err, kv.ErrKeyNotFound) {
		l.Fatalf("expected deleted key=%s to be missing, got: %v", key, err)
	}

	if err := db.Put(key, val); err != nil {
		l.Fatalf("failed to set the key=%s: %v", key, err)
	}
}
