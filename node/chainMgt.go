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
)

func (n *Node) chainMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var ok bool
	var err error
	var multiaddr string
	tick := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-tick.C:
			ok, err = n.NetListening()
			if !ok || err != nil {
				n.SetChainState(false)
				n.Reconnect()
			}
		case filetag := <-n.GetServiceTagCh():
			configs.Tip(fmt.Sprintf("Received a service file tag: %s", filetag))
		case discoverPeer := <-n.DiscoveredPeer():
			multiaddr = fmt.Sprintf("%s/p2p/%s", discoverPeer.Addr.String(), discoverPeer.PeerID.Pretty())
			configs.Tip(fmt.Sprintf("Found a peer: %s", multiaddr))
			_, err = n.AddMultiaddrToPearstore(multiaddr, time.Hour)
			if err != nil {
				configs.Warn(fmt.Sprintf("Add %s to pearstore err: %v", multiaddr, err))
			} else {
				configs.Tip(fmt.Sprintf("Add %s to pearstore", multiaddr))
				n.PutPeer(discoverPeer.PeerID.Pretty(), discoverPeer.Addr.String())
			}
		}
	}
}
