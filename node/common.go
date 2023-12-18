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
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	ma "github.com/multiformats/go-multiaddr"
)

type DataDir struct {
	DbDir     string
	LogDir    string
	SpaceDir  string
	TagDir    string
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
	// Record the fid of stored files
	Cach_prefix_File = "file:"
	// Record the block of reported tags
	Cach_prefix_Tag = "tag:"

	Cach_prefix_MyLost      = "mylost:"
	Cach_prefix_recovery    = "recovery:"
	Cach_prefix_TargetMiner = "targetminer:"
	Cach_prefix_ParseBlock  = "parseblocks"
)

func (n *Node) connectBoot() {
	chainSt := n.GetChainState()
	if chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

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
			n.Connect(n.GetCtxQueryFromCtxCancel(), *addrInfo)
			n.GetDht().RoutingTable().TryAddPeer(addrInfo.ID, true, true)
		}
	}
}

func (n *Node) connectChain(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt == pattern.MINER_STATE_EXIT ||
		minerSt == pattern.MINER_STATE_OFFLINE {
		return
	}

	n.Log("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	n.Ichal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	n.Schal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	out.Err(fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	err := n.ReconnectRPC()
	if err != nil {
		n.Log("err", "All RPCs failed to reconnect")
		n.Ichal("err", "All RPCs failed to reconnect")
		n.Schal("err", "All RPCs failed to reconnect")
		out.Err("All RPCs failed to reconnect")
		return
	}
	n.SetChainState(true)
	out.Tip(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
}

func (n *Node) syncChainStatus(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	teelist, err := n.QueryAllTeeInfo()
	if err != nil {
		n.Log("err", err.Error())
	} else {
		for i := 0; i < len(teelist); i++ {
			err = n.SaveTee(teelist[i].WorkAccount, teelist[i].EndPoint, teelist[i].TeeType)
			if err != nil {
				n.Log("err", err.Error())
			}
		}
	}
	minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
	if err != nil {
		n.Log("err", err.Error())
	} else {
		err = n.SaveMinerState(string(minerInfo.State))
		if err != nil {
			n.Log("err", err.Error())
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
