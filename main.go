package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/1garo/kival/kv"
)

func SaveData2(path string, data []byte) (err error) {
	tmp := fmt.Sprintf("%s.tmp.%d", path, rand.Int())
	fd, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
	if err != nil {
		return err
	}
	defer func() {
		fd.Close()
		if err != nil {
			os.Remove(tmp)
		}
	}()

	if _, err = fd.Write(data); err != nil {
		return err
	}

	abs, _ := filepath.Abs(fd.Name())
	dir := filepath.Dir(abs)
	dirD, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirD.Close()

	if err = dirD.Sync(); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func SaveData1(path string, data []byte) error {
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err := fd.Write(data); err != nil {
		return err
	}
	defer fd.Sync()

	return nil
}

func main() {
	store, err := kv.Open("./data")
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}

	fmt.Println("Store loaded.")

	// Write some values
	if err := store.Put([]byte("name"), []byte("alex")); err != nil {
		log.Fatal(err)
	}

	fmt.Println("data wrote to the db")

	// Read them back
	val, err := store.Get([]byte("name"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("val:", string(val))

	err = store.Del([]byte("name"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("key deleted")
	val, err = store.Get([]byte("name"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("val:", string(val))
}
