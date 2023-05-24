package controllers

import (
	"fmt"

	"github.com/samuel/go-zookeeper/zk"
)

func performLeaderElection(conn *zk.Conn) (bool, error) {

	// Create a ZooKeeper ephemeral node for this client
	nodePath, err := conn.Create("/my-election-node-", []byte{}, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		return false, err
	}

	// Get the list of all ephemeral nodes under the same path

	nodes, _, err := conn.Children("/")
	if err != nil {
		return false, err
	}

	// Determine if this client is the leader based on the smallest sequence number
	isLeader := true
	for _, node := range nodes {
		if node < nodePath {
			isLeader = false
			break
		}
	}

	if isLeader {
		fmt.Println("This client is the leader.")
	} else {
		fmt.Println("This client is not the leader.")
	}

	// Return the result of the leader election
	return isLeader, nil
}
