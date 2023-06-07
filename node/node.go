/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/sdk-go/core/sdk"
)

type Bucket interface {
	Run()
}

type Node struct {
	confile.Confile
	logger.Logger
	cache.Cache
	sdk.SDK
	core.P2P
	Key             *proof.RSAKeyPair
	TeePeerLock     *sync.RWMutex
	StoragePeerLock *sync.RWMutex
	TeePeer         map[string]int64
	StoragePeer     map[string]string
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		Key:             proof.NewKey(),
		TeePeerLock:     new(sync.RWMutex),
		StoragePeerLock: new(sync.RWMutex),
		TeePeer:         make(map[string]int64, 10),
		StoragePeer:     make(map[string]string, 10),
	}
}

func (n *Node) Run() {
	go n.TaskMgt()
	configs.Ok("Start successfully")
	select {}
}

func (n *Node) SaveTeePeer(peerid string, value int64) {
	n.TeePeerLock.Lock()
	defer n.TeePeerLock.Unlock()
	if _, ok := n.TeePeer[peerid]; !ok {
		n.TeePeer[peerid] = value
	}
}

func (n *Node) SaveAndUpdateTeePeer(peerid string, value int64) {
	n.TeePeerLock.Lock()
	defer n.TeePeerLock.Unlock()
	n.TeePeer[peerid] = value
}

func (n *Node) HasTeePeer(peerid string) bool {
	n.TeePeerLock.RLock()
	defer n.TeePeerLock.RUnlock()
	_, ok := n.TeePeer[peerid]
	return ok
}

func (n *Node) GetAllTeePeerId() []string {
	n.TeePeerLock.RLock()
	defer n.TeePeerLock.RUnlock()
	var result = make([]string, len(n.TeePeer))
	var i int
	for k, _ := range n.TeePeer {
		result[i] = k
		i++
	}
	return result
}

func (n *Node) SaveStoragePeer(peerid string, stakingAcc string) {
	n.StoragePeerLock.Lock()
	defer n.StoragePeerLock.Unlock()
	if _, ok := n.StoragePeer[peerid]; !ok {
		n.StoragePeer[peerid] = stakingAcc
	}
}

func (n *Node) HasStoragePeer(peerid string) bool {
	n.StoragePeerLock.RLock()
	defer n.StoragePeerLock.RUnlock()
	_, ok := n.StoragePeer[peerid]
	return ok
}

func (n *Node) deepCopyPeers(dst, src interface{}) error {
	n.TeePeerLock.RLock()
	defer n.TeePeerLock.RUnlock()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}
