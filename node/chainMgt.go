/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func (n *Node) chainMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	tickListening := time.NewTicker(time.Minute)
	defer tickListening.Stop()

	tickConnect := time.NewTicker(time.Hour)
	defer tickConnect.Stop()

	n.Log("info", ">>>>> start chainMgt <<<<<")

	for {
		select {
		case <-tickListening.C:
			if err := n.connectChain(); err != nil {
				n.Log("err", pattern.ERR_RPC_CONNECTION.Error())
				configs.Err(pattern.ERR_RPC_CONNECTION.Error())
				break
			}
			n.syncChainStatus()
		case <-tickConnect.C:
			n.connectBoot()
		}

	}
}

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
		configs.Tip("rpc reconnection successful")
		n.SetChainState(true)
	}
	return nil
}

func (n *Node) syncChainStatus() {
	teelist, err := n.QueryTeeWorkerList()
	if err != nil {
		n.Log("err", fmt.Sprintf("[QueryTeeWorkerList] %v", err))
	} else {
		for _, v := range teelist {
			n.SaveTeeWork(v.Controller_account, v.Peer_id)
		}
	}
}
