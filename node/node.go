/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multiaddr"
)

type Node struct {
	key        *proof.RSAKeyPair
	peerLock   *sync.RWMutex
	teeLock    *sync.RWMutex
	peers      map[string]peer.AddrInfo
	teeWorkers map[string][]byte
	peersPath  string
	sdk.SDK
	confile.Confile
	logger.Logger
	cache.Cache
	*Pois
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		peerLock:   new(sync.RWMutex),
		teeLock:    new(sync.RWMutex),
		peers:      make(map[string]peer.AddrInfo, 0),
		teeWorkers: make(map[string][]byte, 10),
		Pois:       &Pois{},
	}
}

func (n *Node) Run() {
	var (
		ch_spaceMgt         = make(chan bool, 1)
		ch_idlechallenge    = make(chan bool, 1)
		ch_servicechallenge = make(chan bool, 1)
		//ch_restoreMgt  = make(chan bool, 1)
		ch_discoverMgt = make(chan bool, 1)
	)

	ch_idlechallenge <- true
	ch_servicechallenge <- true
	defer close(ch_idlechallenge)
	defer close(ch_servicechallenge)

	// peer persistent location
	n.peersPath = filepath.Join(n.Workspace(), "peers")

	for {
		pubkey, err := n.QueryTeePodr2Puk()
		if err != nil {
			time.Sleep(pattern.BlockInterval)
			continue
		}
		err = n.SetPublickey(pubkey)
		if err != nil {
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Chal("info", "Initialize key successfully")
		break
	}

	task_12S := time.NewTicker(time.Second * 12)
	defer task_12S.Stop()

	task_Minute := time.NewTicker(time.Minute)
	defer task_Minute.Stop()

	task_Hour := time.NewTicker(time.Hour)
	defer task_Hour.Stop()

	// go n.restoreMgt(ch_restoreMgt)
	go n.discoverMgt(ch_discoverMgt)

	// go n.poisMgt(ch_spaceMgt)

	out.Ok("start successfully")

	for {
		select {
		case <-task_12S.C:
			err := n.connectChain()
			if err != nil {
				n.Log("err", pattern.ERR_RPC_CONNECTION.Error())
				out.Err(pattern.ERR_RPC_CONNECTION.Error())
				break
			}

			if len(ch_idlechallenge) > 0 {
				_ = <-ch_idlechallenge
				err := n.poisChallenge(ch_idlechallenge)
				if err != nil {
					n.Chal("err", err.Error())
				}
			}

			if len(ch_servicechallenge) > 0 {
				_ = <-ch_servicechallenge
				err = n.poisServiceChallenge(ch_servicechallenge)
				if err != nil {
					n.Chal("err", err.Error())
				}
			}

		case <-task_Minute.C:
			n.syncChainStatus()
			err := n.reportFiles()
			if err != nil {
				n.Report("err", err.Error())
			}
			err = n.serviceTag()
			if err != nil {
				n.Stag("err", err.Error())
			}
			n.replaceIdle()
		case <-task_Hour.C:
			n.connectBoot()
			if err := n.resizeSpace(); err != nil {
				n.Replace("err", err.Error())
			}
		case <-ch_spaceMgt:
			go n.poisMgt(ch_spaceMgt)
		// case <-ch_restoreMgt:
		// 	go n.restoreMgt(ch_restoreMgt)
		case <-ch_discoverMgt:
			go n.discoverMgt(ch_discoverMgt)
		}
	}
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

func (n *Node) SaveOrUpdatePeerUnSafe(peerid string, addr peer.AddrInfo) {
	n.peers[peerid] = addr
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

func (n *Node) GetAllPeerIdString() []string {
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

func (n *Node) GetAllPeerID() []peer.ID {
	var result = make([]peer.ID, len(n.peers))
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()
	var i int
	for _, v := range n.peers {
		result[i] = v.ID
		i++
	}
	return result
}

func (n *Node) GetAllPeerIDMap() map[string]peer.AddrInfo {
	var result = make(map[string]peer.AddrInfo, len(n.peers))
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()
	for k, v := range n.peers {
		result[k] = v
	}
	return result
}

func (n *Node) RemovePeerIntranetAddr() {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	for k, v := range n.peers {
		var addrInfo peer.AddrInfo
		var addrs []multiaddr.Multiaddr
		for _, addr := range v.Addrs {
			if ipv4, ok := utils.FildIpv4([]byte(addr.String())); ok {
				if ok, err := utils.IsIntranetIpv4(ipv4); err == nil {
					if !ok {
						addrs = append(addrs, addr)
					}
				}
			}
		}
		if len(addrs) > 0 {
			addrInfo.ID = v.ID
			addrInfo.Addrs = utils.RemoveRepeatedAddr(addrs)
			n.SaveOrUpdatePeerUnSafe(v.ID.Pretty(), addrInfo)
		} else {
			delete(n.peers, k)
		}
	}
}

func (n *Node) SavePeersToDisk(path string) error {
	n.peerLock.RLock()
	buf, err := json.Marshal(n.peers)
	if err != nil {
		n.peerLock.RUnlock()
		return err
	}
	n.peerLock.RUnlock()
	err = sutils.WriteBufToFile(buf, n.peersPath)
	return err
}

func (n *Node) LoadPeersFromDisk(path string) error {
	buf, err := os.ReadFile(n.peersPath)
	if err != nil {
		return err
	}
	n.peerLock.Lock()
	err = json.Unmarshal(buf, &n.peers)
	n.peerLock.Unlock()
	return err
}

// tee peers

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

func (n *Node) GetAllTeeWorkPeerIdString() []string {
	var result = make([]string, len(n.teeWorkers))
	n.teeLock.RLock()
	defer n.teeLock.RUnlock()
	var i int
	for _, v := range n.teeWorkers {
		result[i] = base58.Encode(v)
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
