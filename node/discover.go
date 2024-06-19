/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func Subscribe(ctx context.Context, h host.Host, minerRecord MinerRecord, bootnode string) {
	var (
		err      error
		room     string
		findpeer peer.AddrInfo
	)

	gossipSub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return
	}

	data := strings.Split(bootnode, "/p2p/")
	if len(data) > 1 {
		room = fmt.Sprintf("%s-%s", core.NetworkRoom, data[len(data)-1])
	} else {
		room = core.NetworkRoom
	}

	// setup local mDNS discovery
	if err := setupDiscovery(h); err != nil {
		return
	}

	// join the pubsub topic called librum
	topic, err := gossipSub.Join(room)
	if err != nil {
		return
	}

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		return
	}

	out.Ok(fmt.Sprintf("subscribe to a room: %s", room))

	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == h.ID() {
			continue
		}

		err = json.Unmarshal(msg.Data, &findpeer)
		if err != nil {
			continue
		}
		//log.Println("got a peer: ", findpeer.ID.String())
		minerRecord.SavePeer(findpeer)
	}
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.String())
	err := n.h.Connect(context.TODO(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.String(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, "", &discoveryNotifee{h: h})
	return s.Start()
}
