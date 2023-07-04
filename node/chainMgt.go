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
	var err error
	var maAddr ma.Multiaddr
	var addrInfo *peer.AddrInfo

	tickListening := time.NewTicker(time.Minute)
	defer tickListening.Stop()

	n.Log("info", ">>>>> start chainMgt <<<<<")

	for {
		select {
		case <-tickListening.C:
			if !n.GetChainState() {
				err = n.Reconnect()
				if err != nil {
					n.Log("err", pattern.ERR_RPC_CONNECTION.Error())
					configs.Err(pattern.ERR_RPC_CONNECTION.Error())
				} else {
					n.Log("info", "rpc reconnection successful")
					configs.Tip("rpc reconnection successful")
					n.SetChainState(true)
				}
			}

			boots := n.GetBootNodes()
			var bootstrap []string
			for _, b := range boots {
				temp, err := sutils.ParseMultiaddrs(b)
				if err != nil {
					n.Log("err", fmt.Sprintf("[ParseMultiaddrs %v] %v", b, err))
					continue
				}
				bootstrap = append(bootstrap, temp...)
			}
			for _, v := range bootstrap {
				maAddr, err = ma.NewMultiaddr(v)
				if err != nil {
					continue
				}
				addrInfo, err = peer.AddrInfoFromP2pAddr(maAddr)
				if err != nil {
					continue
				}
				err = n.Connect(n.GetRootCtx(), *addrInfo)
				if err != nil {
					continue
				}
				n.SavePeer(addrInfo.ID.Pretty(), *addrInfo)
			}
		}
	}
}
