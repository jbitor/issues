package dht

import (
	"log"
	"os"
)

var logger *log.Logger

// Initializes the logger for this package, using os.Stderr.
func init() {
	logger = log.New(os.Stderr, "[     DHT     ] ", log.Lshortfile)
}
