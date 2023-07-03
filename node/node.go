/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"sync"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
)

type Bucket interface {
	Run()
}

type Node struct {
	confile.Confile
	logger.Logger
	cache.Cache
	sdk.SDK
	key             *proof.RSAKeyPair
	teePeerLock     *sync.RWMutex
	storagePeerLock *sync.RWMutex
	deossPeerLock   *sync.RWMutex
	teePeer         map[string]int64
	storagePeer     map[string]string
	deossPeer       map[string]struct{}
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		teePeerLock:     new(sync.RWMutex),
		storagePeerLock: new(sync.RWMutex),
		deossPeerLock:   new(sync.RWMutex),
		teePeer:         make(map[string]int64, 10),
		storagePeer:     make(map[string]string, 10),
		deossPeer:       make(map[string]struct{}, 10),
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
	n.teePeerLock.Lock()
	defer n.teePeerLock.Unlock()
	if _, ok := n.teePeer[peerid]; !ok {
		n.teePeer[peerid] = value
	}
}

func (n *Node) SaveAndUpdateTeePeer(peerid string, value int64) {
	n.teePeerLock.Lock()
	defer n.teePeerLock.Unlock()
	n.teePeer[peerid] = value
}

func (n *Node) HasTeePeer(peerid string) bool {
	n.teePeerLock.RLock()
	defer n.teePeerLock.RUnlock()
	_, ok := n.teePeer[peerid]
	return ok
}

func (n *Node) GetAllTeePeerId() []string {
	n.teePeerLock.RLock()
	defer n.teePeerLock.RUnlock()
	var result = make([]string, len(n.teePeer))
	var i int
	for k, _ := range n.teePeer {
		result[i] = k
		i++
	}
	return result
}

func (n *Node) SaveStoragePeer(peerid string, stakingAcc string) {
	n.storagePeerLock.Lock()
	defer n.storagePeerLock.Unlock()
	if _, ok := n.storagePeer[peerid]; !ok {
		n.storagePeer[peerid] = stakingAcc
	}
}

func (n *Node) HasStoragePeer(peerid string) bool {
	n.storagePeerLock.RLock()
	defer n.storagePeerLock.RUnlock()
	_, ok := n.storagePeer[peerid]
	return ok
}

func (n *Node) SaveDeossPeer(peerid string) {
	n.deossPeerLock.Lock()
	defer n.deossPeerLock.Unlock()
	if _, ok := n.deossPeer[peerid]; !ok {
		n.deossPeer[peerid] = struct{}{}
	}
}

func (n *Node) HasDeossPeer(peerid string) bool {
	n.deossPeerLock.RLock()
	defer n.deossPeerLock.RUnlock()
	_, ok := n.deossPeer[peerid]
	return ok
}

func (n *Node) RebuildDirs() {
	os.RemoveAll(n.GetDirs().FileDir)
	os.RemoveAll(n.GetDirs().IdleDataDir)
	os.RemoveAll(n.GetDirs().IdleTagDir)
	os.RemoveAll(n.GetDirs().ProofDir)
	os.RemoveAll(n.GetDirs().ServiceTagDir)
	os.RemoveAll(n.GetDirs().TmpDir)
	os.RemoveAll(filepath.Join(n.Workspace(), configs.DbDir))
	os.RemoveAll(filepath.Join(n.Workspace(), configs.LogDir))
	os.MkdirAll(n.GetDirs().FileDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().TmpDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().IdleDataDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().IdleTagDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().ProofDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().ServiceTagDir, pattern.DirMode)
}
