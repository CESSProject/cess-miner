/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	ma "github.com/multiformats/go-multiaddr"
)

type DataDir struct {
	DbDir     string
	LogDir    string
	SpaceDir  string
	PoisDir   string
	AccDir    string
	RandomDir string
	PeersFile string
}

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	Cach_prefix_metadata    = "metadata:"
	Cach_prefix_MyLost      = "mylost:"
	Cach_prefix_recovery    = "recovery:"
	Cach_prefix_TargetMiner = "targetminer:"
	Cach_prefix_File        = "file:"
	Cach_prefix_ParseBlock  = "parseblocks"
)

func (n *Node) connectBoot() {
	boots := n.GetBootNodes()
	for _, b := range boots {
		multiaddr, err := core.ParseMultiaddrs(b)
		if err != nil {
			n.Log("err", fmt.Sprintf("[ParseMultiaddrs %v] %v", b, err))
			continue
		}
		for _, v := range multiaddr {
			maAddr, err := ma.NewMultiaddr(v)
			if err != nil {
				continue
			}
			addrInfo, err := peer.AddrInfoFromP2pAddr(maAddr)
			if err != nil {
				continue
			}
			err = n.Connect(n.GetCtxQueryFromCtxCancel(), *addrInfo)
			if err != nil {
				continue
			}
			n.SavePeer(addrInfo.ID.Pretty(), *addrInfo)
		}
	}
}

func (n *Node) connectChain() error {
	var err error
	if !n.GetChainState() {
		n.Log("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
		n.Ichal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
		n.Schal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
		out.Err(fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
		err = n.Reconnect()
		if err != nil {
			return err
		}
		out.Tip(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
		n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
		n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
		n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
		n.SetChainState(true)
	}
	return nil
}

func (n *Node) syncChainStatus() {
	teelist, err := n.QueryTeeWorkerList()
	if err != nil {
		n.Log("err", fmt.Sprintf("[QueryTeeWorkerList] %v", err))
	} else {
		for i := 0; i < len(teelist); i++ {
			n.SaveTeeWork(teelist[i].Controller_account, teelist[i].Peer_id)
		}
	}
}

func (n *Node) watchMem() {
	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(time.Second * 3)
	defer tikProgram.Stop()

	for {
		select {
		case <-tikProgram.C:
			runtime.ReadMemStats(memSt)
			if memSt.HeapSys >= pattern.SIZE_1GiB*8 {
				n.Log("err", fmt.Sprintf("Mem heigh: %d", memSt.HeapSys))
				os.Exit(1)
			}
		}
	}
}
