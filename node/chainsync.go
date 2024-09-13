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

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/utils"
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

const (
	Unregistered = iota
	UnregisteredPoisKey
	Registered
)

func (n *Node) SyncTeeInfo(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var dialOptions []grpc.DialOption
	var chainPublickey = make([]byte, chain.WorkerPublicKeyLen)
	teelist, err := n.QueryAllWorkers(-1)
	if err != nil {
		n.Log("err", err.Error())
		return
	}
	for i := 0; i < len(teelist); i++ {
		n.Log("info", fmt.Sprintf("check tee: %s", hex.EncodeToString([]byte(string(teelist[i].Pubkey[:])))))
		endpoint, err := n.QueryEndpoints(teelist[i].Pubkey, -1)
		if err != nil {
			n.Log("err", err.Error())
			continue
		}
		endpoint = ProcessTeeEndpoint(endpoint)

		if !strings.Contains(endpoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}

		// verify identity public key
		identityPubkeyResponse, err := com.GetIdentityPubkey(endpoint,
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
		if len(identityPubkeyResponse.Pubkey) != chain.WorkerPublicKeyLen {
			n.DeleteTee(string(teelist[i].Pubkey[:]))
			n.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey length err: %d", len(identityPubkeyResponse.Pubkey)))
			continue
		}

		for j := 0; j < chain.WorkerPublicKeyLen; j++ {
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

func (n *Node) syncMinerStatus() {
	minerInfo, err := n.QueryMinerItems(n.GetSignatureAccPulickey(), -1)
	if err != nil {
		n.Log("err", err.Error())
		if err.Error() == chain.ERR_Empty {
			n.SetState(chain.MINER_STATE_EXIT)
		}
		return
	}
	n.SetState(string(minerInfo.State))
	acc, err := sutils.EncodePublicKeyAsCessAccount(minerInfo.StakingAccount[:])
	if err == nil {
		n.SetStakingAcc(acc)
	}
	acc, err = sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAccount[:])
	if err == nil {
		n.SetEarningsAcc(acc)
	}
	n.SetSpaceInfo(
		minerInfo.DeclarationSpace.Uint64(),
		minerInfo.IdleSpace.Uint64(),
		minerInfo.ServiceSpace.Uint64(),
		minerInfo.LockSpace.Uint64(),
	)
}

func WatchMem() {
	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(time.Second * 3)
	defer tikProgram.Stop()

	for range tikProgram.C {
		runtime.ReadMemStats(memSt)
		if memSt.HeapSys >= chain.SIZE_1GiB*8 {
			//log("err", fmt.Sprintf("Mem heigh: %d", memSt.HeapSys))
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
