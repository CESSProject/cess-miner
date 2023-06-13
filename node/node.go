/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
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
	key             *proof.RSAKeyPair
	TeePeerLock     *sync.RWMutex
	StoragePeerLock *sync.RWMutex
	teePeer         map[string]int64
	storagePeer     map[string]string
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		TeePeerLock:     new(sync.RWMutex),
		StoragePeerLock: new(sync.RWMutex),
		teePeer:         make(map[string]int64, 10),
		storagePeer:     make(map[string]string, 10),
	}
}

func (n *Node) Run() {
	go n.TaskMgt()
	configs.Ok("Start successfully")
	select {}
}

func (n *Node) SetPublickey(pubkey []byte) error {
	rsaPubkey, err := x509.ParsePKCS1PublicKey(pubkey)
	if err != nil {
		return err
	}
	if n.key == nil {
		n.key = proof.NewKey()
	}
	n.key.Spk = rsaPubkey
	return nil
}

func (n *Node) SaveTeePeer(peerid string, value int64) {
	n.TeePeerLock.Lock()
	defer n.TeePeerLock.Unlock()
	if _, ok := n.teePeer[peerid]; !ok {
		n.teePeer[peerid] = value
	}
}

func (n *Node) SaveAndUpdateTeePeer(peerid string, value int64) {
	n.TeePeerLock.Lock()
	defer n.TeePeerLock.Unlock()
	n.teePeer[peerid] = value
}

func (n *Node) HasTeePeer(peerid string) bool {
	n.TeePeerLock.RLock()
	defer n.TeePeerLock.RUnlock()
	_, ok := n.teePeer[peerid]
	return ok
}

func (n *Node) GetAllTeePeerId() []string {
	n.TeePeerLock.RLock()
	defer n.TeePeerLock.RUnlock()
	var result = make([]string, len(n.teePeer))
	var i int
	for k, _ := range n.teePeer {
		result[i] = k
		i++
	}
	return result
}

func (n *Node) SaveStoragePeer(peerid string, stakingAcc string) {
	n.StoragePeerLock.Lock()
	defer n.StoragePeerLock.Unlock()
	if _, ok := n.storagePeer[peerid]; !ok {
		n.storagePeer[peerid] = stakingAcc
	}
}

func (n *Node) HasStoragePeer(peerid string) bool {
	n.StoragePeerLock.RLock()
	defer n.StoragePeerLock.RUnlock()
	_, ok := n.storagePeer[peerid]
	return ok
}

// func (n *Node) deepCopyPeers(dst, src interface{}) error {
// 	n.TeePeerLock.Lock()
// 	defer n.TeePeerLock.Unlock()
// 	var buf bytes.Buffer
// 	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
// 		return err
// 	}
// 	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
// }
