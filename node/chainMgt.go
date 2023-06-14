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
	var ok bool
	var err error
	var peerid string
	var addr string
	var multiaddr string
	var boots []string
	var teeList []pattern.TeeWorkerInfo
	var deossList []string
	var lastMem uint64
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
			if memSt.HeapSys >= pattern.SIZE_1GiB*4 {
				if memSt.HeapAlloc != lastMem {
					n.Log("err", fmt.Sprintf("memory usage: %d bytes", memSt.HeapAlloc))
				}
				//os.Exit(1)
			}
			lastMem = memSt.HeapAlloc
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
				bootstrap, _ := sutils.ParseMultiaddrs(b)
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
			err := n.Connect(n.GetRootCtx(), discoverPeer)
			if err == nil {
				n.Log("info", fmt.Sprintf("discover and connect to %s", peerid))
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
						n.AddMultiaddrToPeerstore(multiaddr, time.Hour)
						break
					}
				}
			}

			if n.HasStoragePeer(peerid) {
				continue
			}
			if n.HasDeossPeer(peerid) {
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
					configs.Tip(fmt.Sprintf("discovered a tee node: %s", peerid))
					break
				}
			}
			if n.HasTeePeer(peerid) {
				continue
			}

			deossList, err = n.QueryDeossPeerIdList()
			if err != nil {
				continue
			}

			for _, v := range deossList {
				if peerid == v {
					n.SaveDeossPeer(peerid)
					configs.Tip(fmt.Sprintf("discovered a deoss node: %s", peerid))
					break
				}
			}

			if n.HasDeossPeer(peerid) {
				continue
			}

			n.SaveStoragePeer(peerid, "")
			configs.Tip(fmt.Sprintf("discovered a storage node: %s", peerid))
		}
	}
}
