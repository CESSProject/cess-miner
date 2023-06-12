/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
	sutils "github.com/CESSProject/sdk-go/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	ma "github.com/multiformats/go-multiaddr"
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
	var peerid string
	var addr string
	var multiaddr string
	var boots []string
	var teeList []pattern.TeeWorkerInfo

	tickListening := time.NewTicker(time.Second * 30)
	defer tickListening.Stop()

	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(time.Second * 3)
	defer tikProgram.Stop()

	n.Log("info", ">>>>> Start log")

	for {
		select {
		case <-tikProgram.C:
			runtime.ReadMemStats(memSt)
			if memSt.HeapSys >= pattern.SIZE_1GiB*5 {
				n.Log("err", fmt.Sprintf("%d", memSt.HeapSys))
				os.Exit(1)
			}
		case <-tickListening.C:
			ok, err = n.NetListening()
			if !ok || err != nil {
				n.SetChainState(false)
				err = n.Reconnect()
				if err != nil {
					configs.Err(pattern.ERR_RPC_CONNECTION.Error())
				}
			}
			boots = n.GetBootNodes()
			for _, b := range boots {
				bootstrap, _ := utils.ParseMultiaddrs(b)
				for _, v := range bootstrap {
					addr, err := ma.NewMultiaddr(v)
					if err != nil {
						continue
					}
					addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
					if err != nil {
						continue
					}
					n.SaveAndUpdateTeePeer(addrInfo.ID.Pretty(), 0)
				}
			}
			if !n.GetDiscoverSt() {
				n.StartDiscover()
			}
		case discoverPeer := <-n.DiscoveredPeer():
			peerid = discoverPeer.ID.Pretty()
			//configs.Tip(fmt.Sprintf("Found a peer: %s addrs: %v", peerid, discoverPeer.Addrs))
			//configs.Tip(fmt.Sprintf("Found a peer: %s", peerid))
			n.Log("info", fmt.Sprintf("Found a peer: %s", peerid))
			err := n.Connect(n.GetRootCtx(), discoverPeer)
			if err != nil {
				//configs.Err(fmt.Sprintf("Connect to %s failed: %v", peerid, err))
				n.Log("err", fmt.Sprintf("Connect to %s failed: %v", peerid, err))
				continue
			} else {
				//configs.Ok(fmt.Sprintf("Connected to %s", peerid))
				n.Log("info", fmt.Sprintf("Connected to %s", peerid))
			}

			for _, v := range discoverPeer.Addrs {
				addr = v.String()
				temp := strings.Split(addr, "/")
				for _, vv := range temp {
					if sutils.IsIPv4(vv) {
						if vv[len(vv)-1] == byte(49) && vv[len(vv)-3] == byte(48) {
							continue
						}
						multiaddr = fmt.Sprintf("%s/p2p/%s", addr, peerid)
						_, err = n.AddMultiaddrToPeerstore(multiaddr, time.Hour)
						// if err != nil {
						// 	configs.Err(fmt.Sprintf("Add %s to pearstore failed: %v", multiaddr, err))
						// } else {
						// 	configs.Tip(fmt.Sprintf("Add %s to pearstore", multiaddr))
						// }
						break
					}
				}
			}

			if n.HasStoragePeer(peerid) {
				continue
			}
			if n.HasTeePeer(peerid) {
				continue
			}
			teeList, err = n.QueryTeeInfoList()
			if err != nil {
				continue
			}
			for _, v := range teeList {
				if peerid == base58.Encode([]byte(string(v.PeerId[:]))) {
					n.SaveTeePeer(peerid, 0)
					configs.Tip(fmt.Sprintf("Save a tee peer: %s", peerid))
					break
				}
			}
			if !n.HasTeePeer(peerid) {
				n.SaveStoragePeer(peerid, "")
				configs.Tip(fmt.Sprintf("Save a storage peer: %s", peerid))
			}
		}
	}
}
