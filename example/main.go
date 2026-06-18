package main

import (
	l "log"

	"github.com/1garo/kival/kv"
	"github.com/1garo/kival/log"
)

func main() {
	db, err := kv.New(kv.DefaultDBPath, log.WithSyncEveryN(100))
	if err != nil {
		l.Fatalf("failed to open the db: %v", err)
	}

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

	if err := db.Del(key); err != nil {
		l.Fatalf("failed to delete the key=%s: %v", key, err)
	}
	l.Println("successfully delete the key")

	if _, err := db.Get(key); err != nil {
		l.Fatalf("failed to get the key=%s: %v", key, err)
	}

	if err := db.Put(key, val); err != nil {
		l.Fatalf("failed to set the key=%s: %v", key, err)
	}
}
