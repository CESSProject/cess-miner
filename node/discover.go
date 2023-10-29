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
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/time/rate"
)

func (n *Node) findPeers(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Discover("info", ">>>>> start findPeers <<<<<")

	var ok bool
	var err error
	var foundPeer peer.AddrInfo
	var interval time.Duration = 30
	var findInterval time.Duration = 1
	var tick = time.NewTicker(time.Second * findInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			findInterval += interval
			if findInterval > 3600 {
				findInterval = interval
				err = n.SavePeersToDisk(n.DataDir.PeersFile)
				if err != nil {
					n.Discover("err", err.Error())
				}
			}
			tick.Reset(time.Second * findInterval)
			peerChan, err := n.GetRoutingTable().FindPeers(n.GetCtxQueryFromCtxCancel(), n.GetRendezvousVersion())
			if err != nil {
				continue
			}
			ok = true
			for ok {
				select {
				case foundPeer, ok = <-peerChan:
					if !ok {
						break
					}
					if foundPeer.ID == n.ID() {
						continue
					}
					err := n.Connect(n.GetCtxQueryFromCtxCancel(), foundPeer)
					if err != nil {
						n.Peerstore().RemovePeer(foundPeer.ID)
					} else {
						for _, addr := range foundPeer.Addrs {
							n.Peerstore().AddAddr(foundPeer.ID, addr, peerstore.AddressTTL)
						}
						n.SavePeer(foundPeer.ID.Pretty(), peer.AddrInfo{
							ID:    foundPeer.ID,
							Addrs: foundPeer.Addrs,
						})
					}
				}
			}
		}
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
						n.Peerstore().AddAddrs(v.ID, v.Addrs, peerstore.AddressTTL)
						n.SavePeer(v.ID.Pretty(), peer.AddrInfo{
							ID:    v.ID,
							Addrs: v.Addrs,
						})
					}
				}
			}
		}
	}
}

func (n *Node) discoverMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	n.Discover("info", ">>>>> start discoverMgt <<<<<")

	n.UpdatePeers()

	tickDiscover := time.NewTicker(time.Hour)
	defer tickDiscover.Stop()

	var r1 = rate.Every(time.Second * 5)
	var limit = rate.NewLimiter(r1, 1)

	var r2 = rate.Every(time.Minute * 30)
	var printLimit = rate.NewLimiter(r2, 1)
	n.RouteTableFindPeers(0)

	for {
		select {
		case discoveredPeer := <-n.GetDiscoveredPeers():
			if limit.Allow() {
				n.Discover("info", "reset")
				tickDiscover.Reset(time.Minute)
			}
			if len(discoveredPeer.Responses) == 0 {
				break
			}
			for _, v := range discoveredPeer.Responses {
				var addrInfo peer.AddrInfo
				var addrs []multiaddr.Multiaddr
				if v != nil {
					for _, addr := range v.Addrs {
						if !utils.InterfaceIsNIL(addr) {
							if ipv4, ok := utils.FildIpv4([]byte(addr.String())); ok {
								if ok, err := utils.IsIntranetIpv4(ipv4); err == nil {
									if !ok {
										addrs = append(addrs, addr)
									}
								}
							}
						}
					}
				}
				if len(addrs) > 0 {
					addrInfo.ID = v.ID
					addrInfo.Addrs = utils.RemoveRepeatedAddr(addrs)
					n.SavePeer(v.ID.Pretty(), addrInfo)
				}
			}
		case <-tickDiscover.C:
			if printLimit.Allow() {
				n.RemovePeerIntranetAddr()
				err := n.SavePeersToDisk(n.DataDir.PeersFile)
				if err != nil {
					n.Discover("err", err.Error())
				}
				allpeer := n.GetAllPeerIdString()
				for _, v := range allpeer {
					n.Discover("info", fmt.Sprintf("found %s", v))
				}
			}
			n.Discover("info", "RouteTableFindPeers")
			_, err := n.RouteTableFindPeers(len(n.peers) + 20)
			if err != nil {
				n.Discover("err", err.Error())
			}
			n.UpdatePeers()
		}
	}
}

func (n *Node) UpdatePeers() {
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
	if len(reportTaskCh) > 0 {
		_ = <-reportTaskCh
		defer func() {
			reportTaskCh <- true
			if err := recover(); err != nil {
				n.Pnc(utils.RecoverError(err))
			}
		}()
		time.Sleep(time.Second * time.Duration(rand.Intn(300)))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "space.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "schal.log"))
		n.ReportLogs(filepath.Join(n.DataDir.LogDir, "ichal.log"))
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
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	return
}
