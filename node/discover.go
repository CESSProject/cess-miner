/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

var room string

func (n *Node) subscribe(ctx context.Context, ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var (
		err      error
		findpeer peer.AddrInfo
	)

	gossipSub, err := pubsub.NewGossipSub(ctx, n.GetHost())
	if err != nil {
		n.Log("err", fmt.Sprintf("NewGossipSub: %s", err))
		return
	}

	bootnode := n.GetBootnode()

	if strings.Contains(bootnode, "12D3KooWRm2sQg65y2ZgCUksLsjWmKbBtZ4HRRsGLxbN76XTtC8T") {
		room = fmt.Sprintf("%s-12D3KooWRm2sQg65y2ZgCUksLsjWmKbBtZ4HRRsGLxbN76XTtC8T", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWEGeAp1MvvUrBYQtb31FE1LPg7aHsd1LtTXn6cerZTBBd") {
		room = fmt.Sprintf("%s-12D3KooWEGeAp1MvvUrBYQtb31FE1LPg7aHsd1LtTXn6cerZTBBd", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWGDk9JJ5F6UPNuutEKSbHrTXnF5eSn3zKaR27amgU6o9S") {
		room = fmt.Sprintf("%s-12D3KooWGDk9JJ5F6UPNuutEKSbHrTXnF5eSn3zKaR27amgU6o9S", core.NetworkRoom)
	} else {
		room = core.NetworkRoom
	}

	// setup local mDNS discovery
	if err := setupDiscovery(n.GetHost()); err != nil {
		n.Log("err", fmt.Sprintf("setupDiscovery: %s", err))
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

	n.Log("info", fmt.Sprintf("Join room: %s", room))

	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == n.ID() {
			continue
		}

		n.Log("info", fmt.Sprintf("subscribe a peer: %s", findpeer.ID.String()))

		err = json.Unmarshal(msg.Data, &findpeer)
		if err != nil {
			continue
		}

		n.SavePeer(findpeer)
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

func (n *Node) connectBoot() {
	maAddr, err := ma.NewMultiaddr(n.PeerNode.GetBootnode())
	if err != nil {
		return
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(maAddr)
	if err != nil {
		return
	}
	for {
		if n.Network().Connectedness(addrInfo.ID) != network.Connected {
			n.Network().DialPeer(context.TODO(), addrInfo.ID)
		}
		time.Sleep(time.Second * 10)
	}
}

func (n *Node) reportLogsMgt(reportTaskCh chan bool) {
	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	if len(reportTaskCh) > 0 {
		<-reportTaskCh
		defer func() {
			reportTaskCh <- true
			if err := recover(); err != nil {
				n.Pnc(utils.RecoverError(err))
			}
		}()
		time.Sleep(time.Second * time.Duration(rand.Intn(600)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "space.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "stag.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "schal.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "ichal.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "restore.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "panic.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "del.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "log.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "report.log"))
	}
}

func (n *Node) ReportLogs(file string) {
	defer func() {
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	fstat, err := os.Stat(file)
	if err != nil {
		return
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	//
	formFile, err := writer.CreateFormFile("file", fstat.Name())
	if err != nil {
		return
	}

	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = io.Copy(formFile, f)
	if err != nil {
		return
	}

	err = writer.Close()
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%sfeedback/log", configs.DefaultDeossAddr), body)
	if err != nil {
		return
	}

	req.Header.Set("Account", n.GetSignatureAcc())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	client.Transport = utils.GlobalTransport
	_, err = client.Do(req)
	if err != nil {
		return
	}
}

func (n *Node) GetFragmentFromOss(fid string) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(utils.RecoverError(err))
		}
	}()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", configs.DefaultDeossAddr, fid), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Account", n.GetSignatureAcc())
	req.Header.Set("Operation", "download")

	client := &http.Client{}
	client.Transport = utils.GlobalTransport
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed")
	}
	data, err := io.ReadAll(resp.Body)
	return data, err
}
