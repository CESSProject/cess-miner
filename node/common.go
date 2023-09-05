/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/libp2p/go-libp2p/core/peer"
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
		multiaddr, err := sutils.ParseMultiaddrs(b)
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
		err = n.Reconnect()
		if err != nil {
			return err
		}
		n.Log("info", "rpc reconnection successful")
		out.Tip("rpc reconnection successful")
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
