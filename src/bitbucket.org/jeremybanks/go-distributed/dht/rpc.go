package dht

import (
	"bitbucket.org/jeremybanks/go-distributed/bencoding"
	"crypto/rand"
	"fmt"
	"strconv"
	"time"
)

type Query struct {
	TransactionId string
	Remote        *RemoteNode
	Result        chan *bencoding.Dict
	Err           chan error
}

func (local *LocalNode) query(remote *RemoteNode, queryType string, arguments bencoding.Dict) (query *Query) {
	query = new(Query)
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
		return
	}

	query.TransactionId = string(transactionId[:])

	local.OutstandingQueries[query.TransactionId] = query

	message["t"] = bencoding.String(query.TransactionId)

	encodedMessage, err := bencoding.Encode(message)

	if err != nil {
		query.Err <- err
		return
	}

	remote.LastRequestTo = time.Now()

	go func() {
		// XXX:
		// Does this wait longer than necessary to send the packet?
		local.Connection.WriteTo(encodedMessage, &remote.Address)
	}()

	return query
}

func (local *LocalNode) Ping(remote *RemoteNode) (<-chan *bencoding.Dict, <-chan error) {
	pingResult := make(chan *bencoding.Dict)
	pingErr := make(chan error)

	query := local.query(remote, "ping", bencoding.Dict{})

	go func() {
		select {
		case value := <-query.Result:
			remote.Id = NodeId((*value)["id"].(bencoding.String))
			pingResult <- value
		case err := <-query.Err:
			pingErr <- err
		}
	}()

	return pingResult, pingErr
}

func (local *LocalNode) FindNode(remote *RemoteNode, id NodeId) (<-chan []*RemoteNode, <-chan error) {
	findResult := make(chan []*RemoteNode)
	findErr := make(chan error)

	query := local.query(remote, "find_node", bencoding.Dict{
		"target": bencoding.String(id),
	})

	go func() {
		select {
		case value := <-query.Result:
			result := []*RemoteNode{}

			fmt.Println("Don't know how to handle:\n", *value)
			findResult <- result
		case err := <-query.Err:
			findErr <- err
		}
	}()

	return findResult, findErr
}

func decodeNodes(local *LocalNode, nodes bencoding.List) []*RemoteNode {
	panic("not implemented")
}

func (local *LocalNode) GetPeers(remote *RemoteNode, id NodeId) (<-chan *bencoding.Dict, <-chan error) {
	panic("not implemented")
}

func (local *LocalNode) AnnouncePeer(remote *RemoteNode, id NodeId) (<-chan *bencoding.Dict, <-chan error) {
	panic("not implemented")
}

func (local *LocalNode) RunRpcListen(rpcError chan<- error) {
	response := new([1024]byte)

	for {
		n, remoteAddr, err := local.Connection.ReadFromUDP(response[:])

		fmt.Printf("Got UDP packet!\n")

		if err != nil {
			fmt.Printf("Ignoring UDP read err: %v\n", err)
			continue
		}

		fmt.Printf("Got response?! %v from %v\n", strconv.Quote(string(response[:n])), remoteAddr)

		result, err := bencoding.Decode(response[:n])

		if err != nil {
			fmt.Printf("Ignoring un-bedecodable message: %v\n", err)
			continue
		}

		resultD, ok := result.(bencoding.Dict)

		if !ok {
			fmt.Printf("Ignoring bedecoded non-dict message: %v\n", err)
			continue
		}

		transactionId := string(resultD["t"].(bencoding.String))

		query := local.OutstandingQueries[transactionId]

		query.Remote.LastResponseFrom = time.Now()

		resultBody, ok := resultD["r"].(bencoding.Dict)

		if !ok {
			fmt.Printf("Ignoring response with non-dict contents.\n")
			continue
		}

		query.Result <- &resultBody
	}

}
