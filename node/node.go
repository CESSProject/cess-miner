/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
)

type Node struct {
	sdk.SDK
	core.P2P
	confile.Confile
	logger.Logger
	cache.Cache
	TeeRecord
	MinerState
	PeerRecord
	*proof.RSAKeyPair
	*pb.MinerPoisInfo
	*DataDir
	*Pois
	peersFile string
	cpuCore   int
}

// New is used to build a node instance
func New() *Node {
	return &Node{
		RSAKeyPair: proof.NewKey(),
		TeeRecord:  NewTeeRecord(),
		MinerState: NewMinerState(),
		PeerRecord: NewPeerRecord(),
		Pois:       &Pois{},
	}
}

func (n *Node) Run() {
	var (
		ch_ConnectChain     = make(chan bool, 1)
		ch_findPeers        = make(chan bool, 1)
		ch_recvPeers        = make(chan bool, 1)
		ch_syncChainStatus  = make(chan bool, 1)
		ch_spaceMgt         = make(chan bool, 1)
		ch_idlechallenge    = make(chan bool, 1)
		ch_servicechallenge = make(chan bool, 1)
		ch_reportfiles      = make(chan bool, 1)
		ch_calctag          = make(chan bool, 1)
		ch_replace          = make(chan bool, 1)
		ch_restoreMgt       = make(chan bool, 1)
		ch_reportLogs       = make(chan bool, 1)
		ch_GenIdleFile      = make(chan bool, 1)
	)
	ch_calctag <- true
	ch_ConnectChain <- true
	ch_idlechallenge <- true
	ch_servicechallenge <- true
	ch_reportfiles <- true
	ch_replace <- true
	ch_reportLogs <- true
	ch_GenIdleFile <- true
	ch_restoreMgt <- true

	for {
		pubkey, err := n.QueryTeePodr2Puk()
		if err != nil {
			time.Sleep(pattern.BlockInterval)
			continue
		}
		err = n.SetPublickey(pubkey)
		if err != nil {
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Schal("info", "Initialize key successfully")
		break
	}

	task_10S := time.NewTicker(time.Duration(time.Second * 10))
	defer task_10S.Stop()

	task_30S := time.NewTicker(time.Duration(time.Second * 30))
	defer task_30S.Stop()

	task_Minute := time.NewTicker(time.Minute)
	defer task_Minute.Stop()

	task_Hour := time.NewTicker(time.Hour)
	defer task_Hour.Stop()

	n.syncChainStatus(ch_syncChainStatus)
	if n.GetMinerState() == pattern.MINER_STATE_FROZEN {
		out.Warn("You are in frozen status, please increase your stake.")
	}

	go n.poisMgt(ch_spaceMgt)
	go n.findPeers(ch_findPeers)
	go n.recvPeers(ch_recvPeers)

	n.Log("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Log("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Ichal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCore()))
	n.Schal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))

	out.Ok("Start successfully")

	for {
		select {
		case <-task_10S.C:
			if len(ch_ConnectChain) > 0 {
				_ = <-ch_ConnectChain
				go n.connectChain(ch_ConnectChain)
			}

		case <-task_30S.C:
			if len(ch_reportfiles) > 0 {
				_ = <-ch_reportfiles
				go n.reportFiles(ch_reportfiles)
			}
			if len(ch_calctag) > 0 {
				_ = <-ch_calctag
				go n.serviceTag(ch_calctag)
			}

		case <-task_Minute.C:
			if len(ch_syncChainStatus) > 0 {
				_ = <-ch_syncChainStatus
				go n.syncChainStatus(ch_syncChainStatus)
			}

			if len(ch_idlechallenge) > 0 || len(ch_servicechallenge) > 0 {
				go n.challengeMgt(ch_idlechallenge, ch_servicechallenge)
			}

			if len(ch_findPeers) > 0 {
				_ = <-ch_findPeers
				go n.findPeers(ch_findPeers)
			}

			if len(ch_recvPeers) > 0 {
				_ = <-ch_recvPeers
				go n.recvPeers(ch_recvPeers)
			}

			if len(ch_GenIdleFile) > 0 {
				_ = <-ch_GenIdleFile
				go n.genIdlefile(ch_GenIdleFile)
			}

			if len(ch_replace) > 0 {
				_ = <-ch_replace
				go n.replaceIdle(ch_replace)
			}

			if len(ch_spaceMgt) > 0 {
				_ = <-ch_spaceMgt
				go n.poisMgt(ch_spaceMgt)
			}

			if len(ch_restoreMgt) > 0 {
				_ = <-ch_restoreMgt
				go n.restoreMgt(ch_restoreMgt)
			}
		case <-task_Hour.C:
			go n.connectBoot()
			// go n.UpdatePeers()
			go n.reportLogsMgt(ch_reportLogs)
		default:
			time.Sleep(time.Second)
		}
	}
}

func (n *Node) SaveCpuCore(cores int) {
	n.cpuCore = cores
}

func (n *Node) GetCpuCore() int {
	return n.cpuCore
}

func (n *Node) GetPodr2Key() *proof.RSAKeyPair {
	return n.RSAKeyPair
}

func (n *Node) SetPublickey(pubkey []byte) error {
	rsaPubkey, err := x509.ParsePKCS1PublicKey(pubkey)
	if err != nil {
		return err
	}
	if n.RSAKeyPair == nil {
		n.RSAKeyPair = proof.NewKey()
	}
	n.RSAKeyPair.Spk = rsaPubkey
	return nil
}

func (n *Node) RebuildDirs() {
	os.RemoveAll(n.GetDirs().FileDir)
	os.RemoveAll(n.GetDirs().TmpDir)
	os.RemoveAll(n.DataDir.DbDir)
	os.RemoveAll(n.DataDir.LogDir)
	os.RemoveAll(n.DataDir.SpaceDir)
	os.RemoveAll(n.DataDir.TagDir)
	os.RemoveAll(n.DataDir.AccDir)
	os.RemoveAll(n.DataDir.PoisDir)
	os.RemoveAll(n.DataDir.RandomDir)
	os.MkdirAll(n.GetDirs().FileDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().TmpDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.TagDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.DbDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.LogDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.SpaceDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.AccDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.PoisDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.RandomDir, pattern.DirMode)
}
