/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/multiformats/go-multiaddr"
)

type Node struct {
	key           *proof.RSAKeyPair
	peerLock      *sync.RWMutex
	teeLock       *sync.RWMutex
	DataDir       *DataDir
	MinerPoisInfo *pb.MinerPoisInfo
	peers         map[string]peer.AddrInfo
	teeWorkers    map[string]string
	peersFile     string
	state         *atomic.Value
	cpuCore       int
	sdk.SDK
	core.P2P
	confile.Confile
	logger.Logger
	cache.Cache
	*Pois
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		key:        proof.NewKey(),
		peerLock:   new(sync.RWMutex),
		teeLock:    new(sync.RWMutex),
		state:      new(atomic.Value),
		peers:      make(map[string]peer.AddrInfo, 0),
		teeWorkers: make(map[string]string, 10),
		Pois:       &Pois{},
	}
}

func (n *Node) Run() {
	var (
		ch_findPeers = make(chan bool, 1)
		ch_recvPeers = make(chan bool, 1)

		ch_spaceMgt         = make(chan bool, 1)
		ch_idlechallenge    = make(chan bool, 1)
		ch_servicechallenge = make(chan bool, 1)
		ch_reportfiles      = make(chan bool, 1)
		ch_calctag          = make(chan bool, 1)
		ch_replace          = make(chan bool, 1)
		ch_restoreMgt       = make(chan bool, 1)
		ch_reportLogs       = make(chan bool, 1)
		ch_GenIdleFile      = make(chan bool, 1)
	)

	ch_idlechallenge <- true
	ch_servicechallenge <- true
	ch_reportfiles <- true
	ch_replace <- true
	ch_reportLogs <- true
	ch_GenIdleFile <- true

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
		n.Schal("info", "Initialize key successfully")
		break
	}

	task_10S := time.NewTicker(time.Duration(time.Second * 10))
	defer task_10S.Stop()

	task_30S := time.NewTicker(time.Duration(time.Second * 30))
	defer task_30S.Stop()

	task_Minute := time.NewTicker(time.Minute)
	defer task_Minute.Stop()

	task_Hour := time.NewTicker(time.Hour)
	defer task_Hour.Stop()

	n.syncChainStatus()

	// go n.watchMem()
	go n.restoreMgt(ch_restoreMgt)
	go n.poisMgt(ch_spaceMgt)
	go n.reportLogsMgt(ch_reportLogs)
	go n.serviceTag(ch_calctag)

	go n.findPeers(ch_findPeers)
	go n.recvPeers(ch_recvPeers)

	n.Log("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Log("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Ichal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Schal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))

	out.Ok("Start successfully")

	for {
		select {
		case <-task_10S.C:
			err := n.connectChain()
			if err != nil {
				n.Log("err", pattern.ERR_RPC_CONNECTION.Error())
				n.Ichal("err", pattern.ERR_RPC_CONNECTION.Error())
				n.Schal("err", pattern.ERR_RPC_CONNECTION.Error())
				out.Err(pattern.ERR_RPC_CONNECTION.Error())
				break
			}

		case <-task_30S.C:
			if len(ch_reportfiles) > 0 {
				_ = <-ch_reportfiles
				go n.reportFiles(ch_reportfiles)
			}
			if len(ch_calctag) > 0 {
				_ = <-ch_calctag
				go n.serviceTag(ch_calctag)
			}

		case <-task_Minute.C:
			n.syncChainStatus()

			if len(ch_idlechallenge) > 0 || len(ch_servicechallenge) > 0 {
				go n.challengeMgt(ch_idlechallenge, ch_servicechallenge)
			}

			if len(ch_findPeers) > 0 {
				_ = <-ch_findPeers
				go n.findPeers(ch_findPeers)
			}
			if len(ch_recvPeers) > 0 {
				_ = <-ch_recvPeers
				go n.recvPeers(ch_recvPeers)
			}
			if len(ch_GenIdleFile) > 0 {
				_ = <-ch_GenIdleFile
				go n.genIdlefile(ch_GenIdleFile)
			}
			if len(ch_replace) > 0 {
				_ = <-ch_replace
				go n.replaceIdle(ch_replace)
			}

		case <-task_Hour.C:
			go n.connectBoot()
			// go n.UpdatePeers()
			go n.reportLogsMgt(ch_reportLogs)

		case <-ch_spaceMgt:
			go n.poisMgt(ch_spaceMgt)

		case <-ch_restoreMgt:
			go n.restoreMgt(ch_restoreMgt)
		}
	}
}

func (n *Node) SaveCpuCore(cores int) {
	n.cpuCore = cores
}

func (n *Node) GetCpuCore() int {
	return n.cpuCore
}

func (n *Node) GetPodr2Key() *proof.RSAKeyPair {
	return n.key
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
	n.peerLock.Lock()
	n.peers[peerid] = addr
	n.peerLock.Unlock()
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
	err = sutils.WriteBufToFile(buf, n.DataDir.PeersFile)
	return err
}

func (n *Node) LoadPeersFromDisk(path string) error {
	buf, err := os.ReadFile(n.DataDir.PeersFile)
	if err != nil {
		return err
	}
	n.peerLock.Lock()
	err = json.Unmarshal(buf, &n.peers)
	n.peerLock.Unlock()
	return err
}

// tee peers

func (n *Node) SaveTeeWork(account, endpoint string) {
	n.teeLock.Lock()
	n.teeWorkers[account] = endpoint
	n.teeLock.Unlock()
}

func (n *Node) GetTeeWork(account string) (string, bool) {
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

func (n *Node) GetAllTeeWorkEndPoint() []string {
	var result = make([]string, len(n.teeWorkers))
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
	os.RemoveAll(n.GetDirs().TmpDir)
	os.RemoveAll(n.DataDir.DbDir)
	os.RemoveAll(n.DataDir.LogDir)
	os.RemoveAll(n.DataDir.SpaceDir)
	os.RemoveAll(n.DataDir.TagDir)
	os.RemoveAll(n.DataDir.AccDir)
	os.RemoveAll(n.DataDir.PoisDir)
	os.RemoveAll(n.DataDir.RandomDir)
	os.MkdirAll(n.GetDirs().FileDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().TmpDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.TagDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.DbDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.LogDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.SpaceDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.AccDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.PoisDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.RandomDir, pattern.DirMode)
}
