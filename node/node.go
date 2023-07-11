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
	"github.com/CESSProject/p2p-go/out"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Bucket interface {
	Run()
}

type Node struct {
	confile.Confile
	logger.Logger
	cache.Cache
	sdk.SDK
	key        *proof.RSAKeyPair
	peerLock   *sync.RWMutex
	teeLock    *sync.RWMutex
	peers      map[string]peer.AddrInfo
	teeWorkers map[string][]byte
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		peerLock:   new(sync.RWMutex),
		teeLock:    new(sync.RWMutex),
		peers:      make(map[string]peer.AddrInfo, 20),
		teeWorkers: make(map[string][]byte, 20),
	}
}

func (n *Node) Run() {
	go n.TaskMgt()
	out.Ok("start successfully")
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

func (n *Node) SavePeer(peerid string, addr peer.AddrInfo) {
	if n.peerLock.TryLock() {
		n.peers[peerid] = addr
		n.peerLock.Unlock()
	}
}

func (n *Node) HasPeer(peerid string) bool {
	n.peerLock.RLock()
	_, ok := n.peers[peerid]
	n.peerLock.RUnlock()
	return ok
}

func (n *Node) GetPeer(peerid string) (peer.AddrInfo, bool) {
	n.peerLock.RLock()
	result, ok := n.peers[peerid]
	n.peerLock.RUnlock()
	return result, ok
}

func (n *Node) GetAllPeerId() []string {
	var result = make([]string, len(n.peers))
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()
	var i int
	for k, _ := range n.peers {
		result[i] = k
		i++
	}
	return result
}

func (n *Node) SaveTeeWork(account string, peerid []byte) {
	if n.teeLock.TryLock() {
		n.teeWorkers[account] = peerid
		n.teeLock.Unlock()
	}
}

func (n *Node) GetTeeWork(account string) ([]byte, bool) {
	n.teeLock.RLock()
	result, ok := n.teeWorkers[account]
	n.teeLock.RUnlock()
	return result, ok
}

func (n *Node) GetAllTeeWorkAccount() []string {
	var result = make([]string, len(n.teeWorkers))
	n.teeLock.RLock()
	defer n.teeLock.RUnlock()
	var i int
	for k, _ := range n.teeWorkers {
		result[i] = k
		i++
	}
	return result
}

func (n *Node) GetAllTeeWorkPeerId() [][]byte {
	var result = make([][]byte, len(n.teeWorkers))
	n.teeLock.RLock()
	defer n.teeLock.RUnlock()
	var i int
	for _, v := range n.teeWorkers {
		result[i] = v
		i++
	}
	return result
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
