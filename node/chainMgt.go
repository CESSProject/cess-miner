/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
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
	var err error
	var peerid string
	var maAddr ma.Multiaddr
	var addrInfo *peer.AddrInfo
	var boots []string
	var teeList []pattern.TeeWorkerInfo
	var deossList []string
	var bootstrap []string
	var lastMem uint64
	tickListening := time.NewTicker(time.Minute)
	defer tickListening.Stop()

	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(pattern.BlockInterval)
	defer tikProgram.Stop()

	n.Log("info", ">>>>> Start chainMgt")

	for {
		select {
		case <-tikProgram.C:
			runtime.ReadMemStats(memSt)
			if memSt.HeapSys >= pattern.SIZE_1GiB*2 {
				if memSt.HeapAlloc != lastMem {
					n.Log("err", fmt.Sprintf("memory usage: %d bytes", memSt.HeapAlloc))
				}
				//os.Exit(1)
			}
			lastMem = memSt.HeapAlloc
			if !n.GetChainState() {
				err = n.Reconnect()
				if err != nil {
					n.Log("err", pattern.ERR_RPC_CONNECTION.Error())
					configs.Err(pattern.ERR_RPC_CONNECTION.Error())
				} else {
					n.Log("info", "rpc reconnection successful")
					configs.Tip("rpc reconnection successful")
					n.SetChainState(true)
				}
			}
		case <-tickListening.C:
			boots = n.GetBootNodes()
			for _, b := range boots {
				bootstrap, err = sutils.ParseMultiaddrs(b)
				if err != nil {
					n.Log("err", fmt.Sprintf("[ParseMultiaddrs %v] %v", b, err))
					continue
				}
				for _, v := range bootstrap {
					maAddr, err = ma.NewMultiaddr(v)
					if err != nil {
						continue
					}
					addrInfo, err = peer.AddrInfoFromP2pAddr(maAddr)
					if err != nil {
						continue
					}
					err = n.Connect(n.GetRootCtx(), *addrInfo)
					if err != nil {
						n.Log("err", err.Error())
						continue
					}
					n.SaveTeePeer(addrInfo.ID.Pretty(), 0)
				}
			}
			if !n.GetDiscoverSt() {
				n.StartDiscover()
			}
		case discoverPeer := <-n.DiscoveredPeer():
			peerid = discoverPeer.ID.Pretty()
			err = n.Connect(n.GetRootCtx(), discoverPeer)
			if err == nil {
				n.Log("info", fmt.Sprintf("discover and connect to %s", peerid))
			}

			for _, v := range discoverPeer.Addrs {
				boots = strings.Split(v.String(), "/")
				for _, vv := range boots {
					if sutils.IsIPv4(vv) {
						if vv[len(vv)-1] == byte(49) && vv[len(vv)-3] == byte(48) {
							continue
						}
						n.AddMultiaddrToPeerstore(fmt.Sprintf("%s/p2p/%s", v.String(), peerid), time.Hour)
						break
					}
				}
			}

			if n.HasStoragePeer(peerid) {
				break
			}
			if n.HasDeossPeer(peerid) {
				break
			}
			if n.HasTeePeer(peerid) {
				break
			}

			teeList, err = n.QueryTeeInfoList()
			if err != nil {
				break
			}
			for _, v := range teeList {
				if peerid == base58.Encode([]byte(string(v.PeerId[:]))) {
					n.SaveTeePeer(peerid, 0)
					configs.Tip(fmt.Sprintf("discovered a tee node: %s", peerid))
					break
				}
			}
			if n.HasTeePeer(peerid) {
				break
			}

			deossList, err = n.QueryDeossPeerIdList()
			if err != nil {
				break
			}

			for _, v := range deossList {
				if peerid == v {
					n.SaveDeossPeer(peerid)
					configs.Tip(fmt.Sprintf("discovered a deoss node: %s", peerid))
					break
				}
			}

			if n.HasDeossPeer(peerid) {
				break
			}

			n.SaveStoragePeer(peerid, "")
			configs.Tip(fmt.Sprintf("discovered a storage node: %s", peerid))
		}
		time.Sleep(time.Millisecond * 10)
	}
}
