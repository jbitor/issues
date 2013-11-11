package cli

import (
	"crypto/sha1"
	"github.com/jeremybanks/go-distributed/bencoding"
	"github.com/jeremybanks/go-distributed/torrent"
	"github.com/jeremybanks/go-distributed/utils"
	"os"
)

func cmdTorrent(args []string) {
	if len(args) == 0 {
		logger.Fatalf("Usage: %v torrent SUBCOMMAND\n", os.Args[0])
		return
	}

	subcommand := args[0]
	subcommandArgs := args[1:]

	switch subcommand {
	case "make":
		cmdTorrentMake(subcommandArgs)
	default:
		logger.Fatalf("Unknown torrent subcommand: %v\n", subcommand)
		return
	}
}

func cmdTorrentMake(args []string) {
	if len(args) != 1 {
		logger.Fatalf("Usage: %v torrent make PATH\n", os.Args[0])
		return
	}

	path := args[0]

	infoDict, err := utils.GenerateTorrentMetaInfo(utils.CreationOptions{
		Path:           path,
		PieceLength:    32768,
		ForceMultiFile: false,
	})
	if err != nil {
		logger.Fatalf("Error generating torrent: %v\n", err)
		return
	}

	infoData, err := bencoding.Encode(infoDict)
	if err != nil {
		logger.Fatalf("Error encoding torrent infodict (for hashing): %v\n", err)
		return
	}

	torrentDict := bencoding.Dict{
		"info": infoDict,
		"nodes": bencoding.List{
			bencoding.List{
				bencoding.String("127.0.0.1"),
				bencoding.Int(6881),
			},
		},
	}

	torrentData, err := bencoding.Encode(torrentDict)

	if err != nil {
		logger.Fatalf("Error encoding torrent data: %v\n", err)
		return
	}

	hasher := sha1.New()
	hasher.Write(infoData)
	hash := hasher.Sum(nil)
	infoHash := torrent.BTID(hash)

	logger.Printf("Generated torrent btih=%v.\n", infoHash)

	os.Stdout.Write(torrentData)
	os.Stdout.Sync()
}
