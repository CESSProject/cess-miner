/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DataDir struct {
	DbDir           string
	LogDir          string
	SpaceDir        string
	PoisDir         string
	AccDir          string
	RandomDir       string
	PeersFile       string
	Podr2PubkeyFile string
}

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	// Record the fid of stored files
	Cach_prefix_File = "file:"
	// Record the block of reported tags
	Cach_prefix_Tag = "tag:"

	Cach_prefix_MyLost      = "mylost:"
	Cach_prefix_recovery    = "recovery:"
	Cach_prefix_TargetMiner = "targetminer:"
	Cach_prefix_ParseBlock  = "parseblocks"
)

// func (n *Node) connectBoot(ch chan bool) {
// 	defer func() {
// 		ch <- true
// 		if err := recover(); err != nil {
// 			n.Pnc(utils.RecoverError(err))
// 		}
// 	}()
// 	minerSt := n.GetMinerState()
// 	if minerSt != pattern.MINER_STATE_POSITIVE &&
// 		minerSt != pattern.MINER_STATE_FROZEN {
// 		return
// 	}

// 	boots := n.GetBootNodes()
// 	for _, b := range boots {
// 		multiaddr, err := core.ParseMultiaddrs(b)
// 		if err != nil {
// 			n.Log("err", fmt.Sprintf("[ParseMultiaddrs %v] %v", b, err))
// 			continue
// 		}
// 		for _, v := range multiaddr {
// 			maAddr, err := ma.NewMultiaddr(v)
// 			if err != nil {
// 				continue
// 			}
// 			addrInfo, err := peer.AddrInfoFromP2pAddr(maAddr)
// 			if err != nil {
// 				continue
// 			}
// 			n.Connect(context.Background(), *addrInfo)
// 			//n.GetDht().RoutingTable().TryAddPeer(addrInfo.ID, true, true)
// 		}
// 	}
// }

func (n *Node) connectChain(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if chainSt {
		return
	}

	n.PeerNode.DisableRecv()

	minerSt := n.GetMinerState()
	if minerSt == pattern.MINER_STATE_EXIT ||
		minerSt == pattern.MINER_STATE_OFFLINE {
		return
	}

	n.Log("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	n.Ichal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	n.Schal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	out.Err(fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	err := n.ReconnectRPC()
	if err != nil {
		n.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
		n.Log("err", "All RPCs failed to reconnect")
		n.Ichal("err", "All RPCs failed to reconnect")
		n.Schal("err", "All RPCs failed to reconnect")
		out.Err("All RPCs failed to reconnect")
		return
	}
	n.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
	n.SetChainState(true)
	n.PeerNode.EnableRecv()
	out.Tip(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
}

func (n *Node) syncChainStatus(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var dialOptions []grpc.DialOption
	var chainPublickey = make([]byte, pattern.WorkerPublicKeyLen)
	teelist, err := n.QueryAllTeeWorkerMap()
	if err != nil {
		n.Log("err", err.Error())
	} else {
		for i := 0; i < len(teelist); i++ {
			n.Log("info", fmt.Sprintf("check tee: %s", hex.EncodeToString([]byte(string(teelist[i].Pubkey[:])))))
			endpoint, err := n.QueryTeeWorkEndpoint(teelist[i].Pubkey)
			if err != nil {
				n.Log("err", err.Error())
				continue
			}
			endpoint = processEndpoint(endpoint)

			if !strings.Contains(endpoint, "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}

			// verify identity public key
			identityPubkeyResponse, err := n.GetIdentityPubkey(endpoint,
				&pb.Request{
					StorageMinerAccountId: n.GetSignatureAccPulickey(),
				},
				time.Duration(time.Minute),
				dialOptions,
				nil,
			)
			if err != nil {
				n.Log("err", err.Error())
				continue
			}
			//n.Log("info", fmt.Sprintf("get identityPubkeyResponse: %v", identityPubkeyResponse.Pubkey))
			if len(identityPubkeyResponse.Pubkey) != pattern.WorkerPublicKeyLen {
				n.DeleteTee(string(teelist[i].Pubkey[:]))
				n.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey length err: %d", len(identityPubkeyResponse.Pubkey)))
				continue
			}

			for j := 0; j < pattern.WorkerPublicKeyLen; j++ {
				chainPublickey[j] = byte(teelist[i].Pubkey[j])
			}
			if !sutils.CompareSlice(identityPubkeyResponse.Pubkey, chainPublickey) {
				n.DeleteTee(string(teelist[i].Pubkey[:]))
				n.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey: %s", hex.EncodeToString(identityPubkeyResponse.Pubkey)))
				n.Log("err", "identityPubkeyResponse.Pubkey err: not qual to chain")
				continue
			}

			n.Log("info", fmt.Sprintf("Save a tee: %s  %d", endpoint, teelist[i].Role))
			err = n.SaveTee(string(teelist[i].Pubkey[:]), endpoint, uint8(teelist[i].Role))
			if err != nil {
				n.Log("err", err.Error())
			}
		}
	}
	minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
	if err != nil {
		n.Log("err", err.Error())
		if err.Error() == pattern.ERR_Empty {
			err = n.SaveMinerState(pattern.MINER_STATE_OFFLINE)
			if err != nil {
				n.Log("err", err.Error())
			}
		}
	} else {
		err = n.SaveMinerState(string(minerInfo.State))
		if err != nil {
			n.Log("err", err.Error())
		}
		n.SaveMinerSpaceInfo(
			minerInfo.DeclarationSpace.Uint64(),
			minerInfo.IdleSpace.Uint64(),
			minerInfo.ServiceSpace.Uint64(),
			minerInfo.LockSpace.Uint64(),
		)
	}
}

func (n *Node) WatchMem() {
	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(time.Second * 3)
	defer tikProgram.Stop()

	for range tikProgram.C {
		runtime.ReadMemStats(memSt)
		if memSt.HeapSys >= pattern.SIZE_1GiB*8 {
			n.Log("err", fmt.Sprintf("Mem heigh: %d", memSt.HeapSys))
			os.Exit(1)
		}
	}
}

func ProcessTeeEndpoint(endPoint string) string {
	var teeEndPoint string
	if strings.HasPrefix(endPoint, "http://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "http://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":80"
		}
	} else if strings.HasPrefix(endPoint, "https://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "https://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":443"
		}
	} else {
		if !strings.Contains(endPoint, ":") {
			teeEndPoint = endPoint + ":80"
		} else {
			teeEndPoint = endPoint
		}
	}
	return teeEndPoint
}
