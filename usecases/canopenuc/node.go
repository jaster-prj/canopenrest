package canopenuc

import (
	canopen "github.com/jormenjanssen/go-canopen"
)

type NodeConfig struct {
	network *canopen.Network
	Node    int
	EdsFile []byte
}

func (nc *NodeConfig) CreateNode() (*canopen.Node, error) {
	// dicObj, err := canopen.DicEDSParse("objdict.eds")
	dicObj, err := canopen.DicEDSParse(nc.EdsFile)
	if err != nil {
		return nil, err
	}
	node := canopen.NewNode(nc.Node, nc.network, dicObj)
	node.Init()
	node.NMTMaster.ListenForHeartbeat()
	return node, nil
}
