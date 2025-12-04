package kv_test

import (
	"errors"
	"testing"

	"github.com/1garo/kival/kv"
)


func TestKV(t *testing.T) {
	db := kv.New()

	db.Set("bar", []byte("baz"))
	val, err := db.Get("bar")
	if err != nil {
		t.Fatal("should find bar key")
	}

	if string(val) != "baz" {
		t.Fatalf("expected %s, received %s", "baz", string(val))
	}

	if err = db.Delete("bar"); err != nil {
		t.Fatal("should be able to delete bar key")
	}

	_, err = db.Get("bar")
	if !errors.Is(err, kv.ErrKeyNotFound){
		t.Fatalf("err should be '%v', found '%v'", kv.ErrKeyNotFound, err)
	}
}
