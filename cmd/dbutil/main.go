package main

import (
	"fmt"
	"os"

	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

func main() {
	args := os.Args

	if len(args) != 4 {
		fmt.Println("Usage: dbutil <db_path> <key> <value>")
		os.Exit(1)
	}

	path := args[1]
	key := args[2]
	value := args[3]

	stor, err := storage.NewRocksStorage(path)
	if err != nil {
		panic(err)
	}
	byteVal, err := byteutils.Hex2Bytes(value)
	if err != nil {
		panic(err)
	}
	stor.Put([]byte(key), byteVal)
}
