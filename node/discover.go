/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) discoverMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	// var err error
	var peerid string

	n.Discover(">>>>> Start discoverMgt <<<<<")
	for {
		select {
		case discoverPeer := <-n.DiscoveredPeer():
			peerid = discoverPeer.ID.Pretty()
			n.Discover(fmt.Sprintf("discovered:  %s", peerid))
			// err = n.P2P.Connect(n.P2P.GetRootCtx(), discoverPeer)
			// if err != nil {
			// 	continue
			// }
			n.SavePeer(peerid, discoverPeer)
			// n.Discover(fmt.Sprintf("connect to %s", peerid))
		}
	}
}
