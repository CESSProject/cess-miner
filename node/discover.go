/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) discoverMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	n.Discover(">>>>> Start discoverMgt <<<<<")
	tickDiscover := time.NewTicker(time.Minute * 5)
	defer tickDiscover.Stop()

	var reset bool
	var length int
	for {
		select {
		case discoverPeer := <-n.DiscoveredPeer():
			if !reset {
				reset = true
				tickDiscover.Reset(time.Minute * 5)
			}
			n.SavePeer(discoverPeer.ID.Pretty(), discoverPeer)
		case <-tickDiscover.C:
			length = 0
			n.RouteTableFindPeers(len(n.peers) + 30)
		default:
			if reset {
				if length != len(n.peers) {
					length = len(n.peers)
					allPeer := n.GetAllPeerId()
					for _, v := range allPeer {
						n.Discover(v)
					}
				}
			}
			reset = false
		}
	}
}
