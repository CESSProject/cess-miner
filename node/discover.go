/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/pkg/errors"
)

func (n *Node) findPeers(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

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

	for foundPeer := range n.GetDiscoveredPeers() {
		for _, v := range foundPeer.Responses {
			if v != nil {
				if len(v.Addrs) > 0 {
					n.SavePeer(peer.AddrInfo{
						ID:    v.ID,
						Addrs: v.Addrs,
					})
					n.GetDht().RoutingTable().TryAddPeer(v.ID, true, true)
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
			n.SavePeer(peer.AddrInfo{
				ID:    onePeer.ID,
				Addrs: onePeer.Addrs,
			})
		}
	}
	return nil
}

func (n *Node) QueryPeerFromOss(peerid string) (peer.AddrInfo, error) {
	data, err := utils.QueryPeers(configs.DefaultDeossAddr)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	var peers = make(map[string]peer.AddrInfo, 0)
	err = json.Unmarshal(data, &peers)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	for k, v := range peers {
		if k == peerid {
			return v, nil
		}
	}
	return peer.AddrInfo{}, errors.New("not found")
}

func (n *Node) reportLogsMgt(reportTaskCh chan bool) {
	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
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
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "stag.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "schal.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "ichal.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "restore.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "panic.log"))
		time.Sleep(time.Second * time.Duration(rand.Intn(120)))
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
