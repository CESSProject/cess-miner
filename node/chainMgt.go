/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sutils "github.com/CESSProject/sdk-go/core/utils"
)

func (n *Node) chainMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var ok bool
	var loopback bool
	var err error
	var peerid string
	var addr string
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
			peerid = discoverPeer.ID.Pretty()
			configs.Tip(fmt.Sprintf("Found a peer: %s addrs: %v", peerid, discoverPeer.Addrs))
			err := n.Connect(n.GetRootCtx(), discoverPeer)
			if err != nil {
				configs.Err(fmt.Sprintf("Failed to connect to node: %s", peerid))
				continue
			}
			n.PutPeer(peerid)
			for _, v := range discoverPeer.Addrs {
				loopback = false
				addr = v.String()
				temp := strings.Split(addr, "/")
				for _, vv := range temp {
					if sutils.IsIPv4(vv) {
						if vv[len(vv)-1] == byte(1) && vv[len(vv)-3] == byte(0) {
							loopback = true
							break
						}
					}
				}

				if loopback {
					continue
				}

				multiaddr = fmt.Sprintf("%s/p2p/%s", addr, peerid)
				_, err = n.AddMultiaddrToPearstore(multiaddr, time.Hour)
				if err != nil {
					configs.Err(fmt.Sprintf("Add %s to pearstore failed: %v", multiaddr, err))
				} else {
					configs.Tip(fmt.Sprintf("Add %s to pearstore", multiaddr))
				}
			}
		}
	}
}
