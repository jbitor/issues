// Package cli provides a command-line interface for functionality in go-distributed.
package cli

import (
	weakrand "math/rand"
	"os"
	"time"
)

func Main() {
	if len(os.Args) == 1 {
		logger.Fatalf("Usage: %v COMMAND\n", os.Args[0])
		return
	}

	command := os.Args[1]
	commandArgs := os.Args[2:]

	weakrand.Seed(time.Now().UTC().UnixNano())

	switch command {
	case "torrent":
		cmdTorrent(commandArgs)
	case "dht":
		cmdDht(commandArgs)
	case "json":
		cmdJson(commandArgs)
	default:
		logger.Fatalf("Unknown command: %v\n", command)
		return
	}
}
