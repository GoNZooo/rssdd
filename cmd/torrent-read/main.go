package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/GoNZooo/rssdd/encoding/bencoding"
)

func main() {
	filename := os.Args[1]
	file, err := os.Open(filename)
	fileReader := bufio.NewReader(file)
	value, err := bencoding.DecodeValue(fileReader)
	if err != nil {
		panic("Unable to decode: " + err.Error())
	}

	switch value.(type) {
	case bencoding.BencodeInt64:
		fmt.Println("int")
	case bencoding.BencodeString:
		fmt.Println("string")
	case bencoding.BencodeList:
		fmt.Println("list")
	case bencoding.BencodeDict:
		for k := range value.(bencoding.BencodeDict) {
			fmt.Printf("%s\n", k)
		}
	}
}
