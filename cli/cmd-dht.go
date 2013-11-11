package cli

import (
	"fmt"
	"github.com/jeremybanks/go-distributed/dht"
	"github.com/jeremybanks/go-distributed/torrent"
	"os"
	"time"
)

func cmdDht(args []string) {
	if len(args) == 0 {
		logger.Fatalf("Usage: %v dht SUBCOMMAND\n", os.Args[0])
		return
	}

	subcommand := args[0]
	subcommandArgs := args[1:]

	switch subcommand {
	case "connect":
		cmdDhtConnect(subcommandArgs)
	case "get-peers":
		cmdDhtGetPeers(subcommandArgs)
	default:
		logger.Fatalf("Unknown dht subcommand: %v\n", subcommand)
		return
	}
}

func cmdDhtConnect(args []string) {
	if len(args) != 0 {
		logger.Fatalf("Usage: %v dht connect.benc\n", os.Args[0])
		return
	}

	client, err := dht.OpenClient(".dht-peer", false)
	if err != nil {
		logger.Fatalf("Unable to open client: %v\n", err)
		return
	}

	for {
		time.Sleep(60 * time.Second)
	}

	client.Close()
}

func cmdDhtGetPeers(args []string) {
	if len(args) != 1 {
		logger.Fatalf("Usage: %v torrent get-peers INFOHASH\n", os.Args[0])
		return
	}

	infoHash, err := torrent.BTIDFromHex(args[0])

	if err != nil {
		logger.Fatalf("Specified string was not a valid hex infohash [%v].\n", err)
		return
	}

	dhtClient, err := dht.OpenClient(".dht-peer", true)
	if err != nil {
		logger.Fatalf("Unable to open .dht-peer: %v\n", err)
		return
	}

	defer dhtClient.Close()

	peers, err := dhtClient.GetPeers(infoHash)

	if err != nil {
		logger.Fatalf("Unable to find peers: %v\n", err)
	}

	logger.Printf("Found peers for %v:\n", infoHash)
	for _, peer := range peers {
		fmt.Println(peer)
	}
}

func cmdDhtHelloWorld(args []string) {
	/*
		var local *dht.localNode

		if len(args) > 0 {

			save := func() {
				// save localNode
				nodeData, err := bencoding.Encode(local)

				if err != nil {
					logger.Fatalf("Error encoding local node state: %v\n", err)
					return
				}

				file.Truncate(0)
				file.WriteAt(nodeData, 0)
				file.Sync()

				if err != nil {
					logger.Fatalf("Error writing local node state: %v\n", err)
					return
				}

				logger.Printf("Saved localNode state to %v.\n", path)
			}

			go func() {
				for {
					time.Sleep(10 * time.Second)
					save()
				}
			}()

			defer save()
		} else {
			local = dht.NewlocalNode()
		}

		terminate := make(chan bool)
		terminated := make(chan error)

		go local.Run(terminate, terminated)

		knownNode := dht.RemoteNodeFromAddress(net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 6881,
		})

		knownNode = local.AddOrGetRemoteNode(knownNode)

		logger.Printf("Hello, I am %v.\n", local)

		infoHash, _ := hex.DecodeString("5497a53543938b77ef660939d3b32e02be7bc213")
		logger.Printf("Trying to GetPeers for %v.\n", infoHash)
		peersResult, nodesResult, errChan := local.GetPeers(knownNode, string(infoHash))

		select {
		case result := <-peersResult:
			logger.Printf("GetPeers got peers: %v\n", result)
		case result := <-nodesResult:
			logger.Printf("GetPeers got nodes: %v\n", result)
		case err := <-errChan:
			logger.Printf("GetPeers had error: %v\n", err)
		}

		if err := <-terminated; err != nil {
			logger.Fatalf("Error in running localNode: %v\n", err)
		}
	*/
}
