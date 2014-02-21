package dht

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/jbitor/bencoding"
	"github.com/jbitor/jbitor/torrent"
	"net"
	"time"
)

type outstandingQuery struct {
	TransactionId string
	Remote        *RemoteNode
	Result        chan *bencoding.Dict
	Err           chan error
}

func decodeNodeAddress(encoded bencoding.String) (addr net.UDPAddr) {
	return net.UDPAddr{
		IP:   net.IPv4(encoded[0], encoded[1], encoded[2], encoded[3]),
		Port: int(encoded[4])<<8 + int(encoded[5]),
	}
}

func encodeNodeAddress(addr net.UDPAddr) (encoded bencoding.String) {
	if addr.Port >= (1<<32) || addr.Port < 0 {
		panic("Port out of bounds?")
	}

	ip4 := addr.IP.To4()

	return bencoding.String([]byte{
		ip4[0],
		ip4[1],
		ip4[2],
		ip4[3],
		byte((addr.Port >> 8) & 0xFF),
		byte(addr.Port & 0xFF),
	})
}

func (local *localNode) sendoutstandingQuery(remote *RemoteNode, queryType string, arguments bencoding.Dict) (query *outstandingQuery) {
	query = new(outstandingQuery)
	query.Result = make(chan *bencoding.Dict)
	query.Err = make(chan error)
	query.Remote = remote

	if arguments == nil {
		arguments = bencoding.Dict{}
	}

	arguments["id"] = bencoding.String(local.Id)

	// XXX: assert that these keys are not already present?
	message := bencoding.Dict{
		"y": bencoding.String("q"),
		"q": bencoding.String(queryType),
		"a": arguments,
	}

	transactionId := new([4]byte)
	if _, err := rand.Read(transactionId[:]); err != nil {
		query.Err <- err
		close(query.Result)
		close(query.Err)
		return
	}

	query.TransactionId = string(transactionId[:])

	local.OutstandingQueries[query.TransactionId] = query

	message["t"] = bencoding.String(query.TransactionId)

	encodedMessage, err := bencoding.Encode(message)

	if err != nil {
		query.Err <- err
		close(query.Result)
		close(query.Err)
		return
	}

	remote.LastRequestTo = time.Now()

	go func() {
		// XXX: Does this wait longer than necessary to send the packet?
		local.Connection.WriteTo(encodedMessage, &remote.Address)
	}()

	return query
}

func (local *localNode) Ping(remote *RemoteNode) (<-chan *bencoding.Dict, <-chan error) {
	pingResult := make(chan *bencoding.Dict)
	pingErr := make(chan error)

	query := local.sendoutstandingQuery(remote, "ping", bencoding.Dict{})

	go func() {
		defer close(pingResult)
		defer close(pingErr)

		select {
		case value := <-query.Result:
			remote.Id = torrent.BTID((*value)["id"].(bencoding.String))

			remote.ConsecutiveFailedQueries = 0

			pingResult <- value

		case err := <-query.Err:
			remote.ConsecutiveFailedQueries++
			pingErr <- err
		}
	}()

	return pingResult, pingErr
}

const peerContactInfoLen = 6
const nodeContactInfoIdLen = 20
const nodeContactInfoLen = 26

func (local *localNode) decodeNodesString(nodesData bencoding.String, source *RemoteNode) ([]*RemoteNode, error) {
	result := make([]*RemoteNode, 0)

	for offset := 0; offset < len(nodesData); offset += nodeContactInfoLen {
		nodeId := nodesData[offset : offset+nodeContactInfoIdLen]
		nodeAddress := decodeNodeAddress(nodesData[offset+nodeContactInfoIdLen : offset+nodeContactInfoLen])

		resultRemote := RemoteNodeFromAddress(nodeAddress)
		resultRemote.Source = source
		resultRemote.Id = torrent.BTID(nodeId)

		resultRemote = local.AddOrGetRemoteNode(resultRemote)

		result = append(result, resultRemote)
	}

	return result, nil
}

func (local *localNode) FindNode(remote *RemoteNode, id torrent.BTID) (<-chan []*RemoteNode, <-chan error) {
	findResult := make(chan []*RemoteNode)
	findErr := make(chan error)

	query := local.sendoutstandingQuery(remote, "find_node", bencoding.Dict{
		"target": bencoding.String(id),
	})

	go func() {
		defer close(findErr)
		defer close(findResult)

		select {
		case value := <-query.Result:
			nodesData, ok := (*value)["nodes"].(bencoding.String)
			if !ok {
				remote.ConsecutiveFailedQueries++
				findErr <- errors.New(".nodes string does not exist")
				return
			}

			result, err := local.decodeNodesString(nodesData, remote)
			if err != nil {
				findErr <- err
				return
			}

			remote.ConsecutiveFailedQueries = 0
			findResult <- result

		case err := <-query.Err:
			remote.ConsecutiveFailedQueries++
			findErr <- err
		}
	}()

	return findResult, findErr
}

func (local *localNode) GetPeers(remote *RemoteNode, infoHash torrent.BTID) (<-chan []*torrent.RemotePeer, <-chan []*RemoteNode, <-chan error) {
	peersResult := make(chan []*torrent.RemotePeer)
	nodesResult := make(chan []*RemoteNode)
	getPeersErr := make(chan error)

	query := local.sendoutstandingQuery(remote, "get_peers", bencoding.Dict{
		"info_hash": bencoding.String(infoHash),
	})

	go func() {
		defer close(peersResult)
		defer close(nodesResult)
		defer close(getPeersErr)

		select {
		case value := <-query.Result:
			peerData, peersOk := (*value)["values"].(bencoding.List)
			nodesData, nodesOk := (*value)["nodes"].(bencoding.String)

			if peersOk {
				result := make([]*torrent.RemotePeer, len(peerData))

				for i, data := range peerData {
					dataStr, ok := data.(bencoding.String)
					if !ok {
						getPeersErr <- errors.New(".values contained non-string")
						remote.ConsecutiveFailedQueries++
						return
					}

					addr, err := torrent.DecodePeerAddress(dataStr)
					if err != nil {
						remote.ConsecutiveFailedQueries++
						getPeersErr <- err
						return
					}

					result[i] = &torrent.RemotePeer{Address: addr}
				}

				remote.ConsecutiveFailedQueries = 0

				peersResult <- result
			} else if nodesOk {
				result, err := local.decodeNodesString(nodesData, remote)
				if err != nil {
					remote.ConsecutiveFailedQueries++
					getPeersErr <- err
					return
				}

				remote.ConsecutiveFailedQueries = 0
				nodesResult <- result
			} else {
				remote.ConsecutiveFailedQueries++
				getPeersErr <- errors.New(fmt.Sprintf("response did not include peer or node list - %v", *value))
			}

		case err := <-query.Err:
			remote.ConsecutiveFailedQueries++
			getPeersErr <- err
		}
	}()

	return peersResult, nodesResult, getPeersErr
}

func (local *localNode) AnnouncePeer(remote *RemoteNode, id torrent.BTID) (result <-chan *bencoding.Dict, err <-chan error) {
	logger.Fatalf("AnnouncePeer() not implemented\n")
	return
}

func (local *localNode) rpcListenLoop(terminate <-chan bool) {
	response := new([1024]byte)

	for {
		logger.Printf("Waiting for next incoming UDP message.\n")

		n, remoteAddr, err := local.Connection.ReadFromUDP(response[:])

		_ = remoteAddr

		if err != nil {
			logger.Printf("Ignoring UDP read err: %v\n", err)
			continue
		}

		result, err := bencoding.Decode(response[:n])

		if err != nil {
			logger.Printf("Ignoring un-bedecodable message: %v\n", err)
			continue
		}

		resultD, ok := result.(bencoding.Dict)

		if !ok {
			logger.Printf("Ignoring bedecoded non-dict message: %v\n", err)
			continue
		}

		transactionId := string(resultD["t"].(bencoding.String))

		query, ok := local.OutstandingQueries[transactionId]
		if !ok {
			logger.Printf("Ignoring query response with unexpected token.\n")
			continue
		}

		query.Remote.LastResponseFrom = time.Now()

		resultBody, ok := resultD["r"].(bencoding.Dict)
		if !ok {
			logger.Printf("Ignoring response with non-dict contents.\n")
			continue
		}

		query.Result <- &resultBody

		delete(local.OutstandingQueries, transactionId)
	}
}
