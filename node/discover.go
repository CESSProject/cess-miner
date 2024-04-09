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
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

func (n *Node) subscribe(ch chan<- bool) {
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

	gossipSub, err := pubsub.NewGossipSub(context.Background(), n.GetHost())
	if err != nil {
		return
	}

	// join the pubsub topic called librum
	topic, err := gossipSub.Join(core.NetworkRoom)
	if err != nil {
		return
	}

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		return
	}

	for {
		msg, err := subscriber.Next(context.Background())
		if err != nil {
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == n.ID() {
			continue
		}

		err = json.Unmarshal(msg.Data, &findpeer)
		if err != nil {
			continue
		}
		err = n.Connect(context.Background(), findpeer)
		if err != nil {
			continue
		}
		n.SavePeer(findpeer)
	}
}

func (n *Node) connectBoot() {
	boots := n.PeerNode.GetBootstraps()
	for {
		for i := 0; i < len(boots); i++ {
			maAddr, err := ma.NewMultiaddr(boots[i])
			if err != nil {
				continue
			}
			addrInfo, err := peer.AddrInfoFromP2pAddr(maAddr)
			if err != nil {
				continue
			}
			n.Connect(context.Background(), *addrInfo)
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
