package bencoding

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "[  bencoding  ] ", log.Lshortfile)
}
