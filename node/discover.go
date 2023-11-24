/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) findPeers(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	err := n.findpeer()
	if err != nil {
		n.Discover("err", err.Error())
	}
}

func (n *Node) recvPeers(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Discover("info", ">>>>> start recvPeers <<<<<")

	for {
		select {
		case foundPeer := <-n.GetDiscoveredPeers():
			for _, v := range foundPeer.Responses {
				if v != nil {
					if len(v.Addrs) > 0 {
						n.SavePeer(v.ID.Pretty(), peer.AddrInfo{
							ID:    v.ID,
							Addrs: v.Addrs,
						})
						n.GetDht().RoutingTable().TryAddPeer(v.ID, true, true)
					}
				}
			}
		}
	}
}

func (n *Node) findpeer() error {
	peerChan, err := n.GetRoutingTable().FindPeers(
		n.GetCtxQueryFromCtxCancel(),
		n.GetRendezvousVersion(),
	)
	if err != nil {
		return err
	}

	for onePeer := range peerChan {
		if onePeer.ID == n.ID() {
			continue
		}
		err := n.Connect(n.GetCtxQueryFromCtxCancel(), onePeer)
		if err != nil {
			n.GetDht().RoutingTable().RemovePeer(onePeer.ID)
		} else {
			n.GetDht().RoutingTable().TryAddPeer(onePeer.ID, true, true)
			n.SavePeer(onePeer.ID.Pretty(), peer.AddrInfo{
				ID:    onePeer.ID,
				Addrs: onePeer.Addrs,
			})
		}
	}
	return nil
}

func (n *Node) UpdatePeers() {
	if n.state.Load() == configs.State_Offline {
		return
	}
	time.Sleep(time.Second * time.Duration(rand.Intn(120)))
	data, err := utils.QueryPeers(configs.DefaultDeossAddr)
	if err != nil {
		n.Discover("err", err.Error())
	} else {
		err = json.Unmarshal(data, &n.peers)
		if err != nil {
			n.Discover("err", err.Error())
		} else {
			err = n.SavePeersToDisk(n.DataDir.PeersFile)
			if err != nil {
				n.Discover("err", err.Error())
			}
		}
	}
}

func (n *Node) UpdatePeerFirst() {
	time.Sleep(time.Second * time.Duration(rand.Intn(30)))
	data, err := utils.QueryPeers(configs.DefaultDeossAddr)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &n.peers)
	if err != nil {
		return
	}
	n.SavePeersToDisk(n.DataDir.PeersFile)
}

func (n *Node) reportLogsMgt(reportTaskCh chan bool) {
	if n.state.Load() == configs.State_Offline {
		return
	}

	if len(reportTaskCh) > 0 {
		_ = <-reportTaskCh
		defer func() {
			reportTaskCh <- true
			if err := recover(); err != nil {
				n.Pnc(utils.RecoverError(err))
			}
		}()
		time.Sleep(time.Second * time.Duration(rand.Intn(600)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "space.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(60)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "schal.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "ichal.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(60)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "panic.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "log.log"))
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

	req, err := http.NewRequest(http.MethodPost, "http://deoss-pub-gateway.cess.cloud/feedback/log", body)
	if err != nil {
		return
	}

	req.Header.Set("Account", n.GetSignatureAcc())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	client.Transport = utils.GlobalTransport
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}
