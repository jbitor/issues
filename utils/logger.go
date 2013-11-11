package utils

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "[torrent utils] ", log.Lshortfile)
}
