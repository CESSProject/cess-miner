/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/time/rate"
)

func (n *Node) discoverMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	n.Discover("info", ">>>>> start discoverMgt <<<<<")

	err := n.LoadPeersFromDisk(n.peersPath)
	if err != nil {
		n.Discover("err", err.Error())
	}
	// data, err := utils.QueryPeers(configs.DefaultDeossAddr)
	// if err != nil {
	// 	n.Discover("err", err.Error())
	// } else {
	// 	err = json.Unmarshal(data, &n.peers)
	// 	if err != nil {
	// 		n.Discover("err", err.Error())
	// 	} else {
	// 		err = n.SavePeersToDisk(n.peersPath)
	// 		if err != nil {
	// 			n.Discover("err", err.Error())
	// 		}
	// 	}
	// }

	tickDiscover := time.NewTicker(time.Minute)
	defer tickDiscover.Stop()

	var r1 = rate.Every(time.Second * 5)
	var limit = rate.NewLimiter(r1, 1)

	var r2 = rate.Every(time.Minute * 30)
	var printLimit = rate.NewLimiter(r2, 1)
	n.RouteTableFindPeers(0)

	for {
		select {
		case discoveredPeer, _ := <-n.GetDiscoveredPeers():
			if limit.Allow() {
				n.Discover("info", "reset")
				tickDiscover.Reset(time.Minute)
			}
			if len(discoveredPeer.Responses) == 0 {
				break
			}
			for _, v := range discoveredPeer.Responses {
				var addrInfo peer.AddrInfo
				var addrs []multiaddr.Multiaddr
				if v != nil {
					for _, addr := range v.Addrs {
						if !utils.InterfaceIsNIL(addr) {
							if ipv4, ok := utils.FildIpv4([]byte(addr.String())); ok {
								if ok, err := utils.IsIntranetIpv4(ipv4); err == nil {
									if !ok {
										addrs = append(addrs, addr)
									}
								}
							}
						}
					}
				}
				if len(addrs) > 0 {
					addrInfo.ID = v.ID
					addrInfo.Addrs = utils.RemoveRepeatedAddr(addrs)
					n.SavePeer(v.ID.Pretty(), addrInfo)
				}
			}
		case <-tickDiscover.C:
			if printLimit.Allow() {
				n.RemovePeerIntranetAddr()
				err = n.SavePeersToDisk(n.peersPath)
				if err != nil {
					n.Discover("err", err.Error())
				}
				allpeer := n.GetAllPeerIdString()
				for _, v := range allpeer {
					n.Discover("info", fmt.Sprintf("found %s", v))
				}
			}
			n.Discover("info", "RouteTableFindPeers")
			_, err := n.RouteTableFindPeers(len(n.peers) + 20)
			if err != nil {
				n.Discover("err", err.Error())
			}
		}
	}
}
