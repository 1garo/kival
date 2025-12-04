package main

import "fmt"

type DB interface {
	Set(key string, data []byte) error
	Get(key string) []byte
	Delete(key string) error
}

func main() {

	fmt.Println("hello world")
}
