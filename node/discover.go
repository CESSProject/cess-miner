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
	tickDiscover := time.NewTicker(time.Minute)
	defer tickDiscover.Stop()

	var r1 = rate.Every(time.Second * 3)
	var limit = rate.NewLimiter(r1, 1)

	var r2 = rate.Every(time.Minute * 10)
	var printLimit = rate.NewLimiter(r2, 1)
	n.RouteTableFindPeers(0)

	for {
		select {
		case peer, _ := <-n.GetDiscoveredPeers():
			if limit.Allow() {
				tickDiscover.Reset(time.Minute)
			}
			if len(peer.Responses) == 0 {
				break
			}
			for _, v := range peer.Responses {
				n.SavePeer(v.ID.Pretty(), *v)
			}
		case <-tickDiscover.C:
			if printLimit.Allow() {
				allpeer := n.GetAllPeerId()
				for _, v := range allpeer {
					n.Discover("info", fmt.Sprintf("found %s", v))
				}
			}
			n.Discover("info", "RouteTableFindPeers")
			_, err := n.RouteTableFindPeers(len(n.peers) + 10)
			if err != nil {
				n.Discover("err", err.Error())
			}
		}
	}
}
