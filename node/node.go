/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/confile"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	confile.Confiler
	logger.Logger
	cache.Cache
	TeeRecorder
	MinerRecord
	runstatus.Runstatus
	workspace.Workspace
	*chain.ChainClient
	*pb.MinerPoisInfo
	*RSAKeyPair
	*Pois
	*gin.Engine
	chain.ExpendersInfo
}

var (
	n    *Node
	once sync.Once
)

func GetNode() *Node {
	once.Do(func() { n = &Node{} })
	return n
}

func InitConfig(cfg confile.Confiler) {
	GetNode().Confiler = cfg
}

func InitWorkspace(ws string) {
	GetNode().Workspace = workspace.NewWorkspace(ws)
}

func InitChainclient(cli *chain.ChainClient) {
	GetNode().ChainClient = cli
}

func InitRSAKeyPair(key *RSAKeyPair) {
	GetNode().RSAKeyPair = key
}

func InitTeeRecord(tees *TeeRecord) {
	GetNode().TeeRecorder = tees
}

func InitMinerPoisInfo(poisInfo *pb.MinerPoisInfo) {
	GetNode().MinerPoisInfo = poisInfo
}

func InitPois(pois *Pois) {
	GetNode().Pois = pois
}

func InitRunstatus(rt runstatus.Runstatus) {
	GetNode().Runstatus = rt
}

func InitLogger(lg logger.Logger) {
	GetNode().Logger = lg
}

func InitCacher(cace cache.Cache) {
	GetNode().Cache = cace
}

func (n *Node) Start() {
	reportFileCh := make(chan bool, 1)
	reportFileCh <- true
	idleChallCh := make(chan bool, 1)
	idleChallCh <- true
	serviceChallCh := make(chan bool, 1)
	serviceChallCh <- true
	replaceIdleCh := make(chan bool, 1)
	replaceIdleCh <- true
	genIdleCh := make(chan bool, 1)
	genIdleCh <- true
	attestationIdleCh := make(chan bool, 1)
	attestationIdleCh <- true
	syncTeeCh := make(chan bool, 1)
	syncTeeCh <- true
	calcTagCh := make(chan bool, 1)
	calcTagCh <- true
	restoreCh := make(chan bool, 1)
	restoreCh <- true

	tick_twoblock := time.NewTicker(chain.BlockInterval * 2)
	defer tick_twoblock.Stop()

	tick_sixblock := time.NewTicker(chain.BlockInterval * 6)
	defer tick_sixblock.Stop()

	tick_Hour := time.NewTicker(time.Second * time.Duration(3597))
	defer tick_Hour.Stop()
	chainState := true

	out.Ok("Service started successfully")
	for {
		select {
		case <-tick_twoblock.C:
			chainState = n.GetRpcState()
			if !chainState {
				go n.Reconnectrpc()
			}

		case <-tick_sixblock.C:
			chainState = n.GetRpcState()
			if !chainState {
				break
			}

			n.syncMinerStatus()
			if n.GetState() == chain.MINER_STATE_EXIT ||
				n.GetState() == chain.MINER_STATE_OFFLINE {
				break
			}

			if len(syncTeeCh) > 0 {
				<-syncTeeCh
				go n.SyncTeeInfo(syncTeeCh)
			}

			if len(reportFileCh) > 0 {
				<-reportFileCh
				go node.ReportFiles(reportFileCh, cli, runtime, l, wspace.GetFileDir(), wspace.GetTmpDir())
			}

			if len(attestationIdleCh) > 0 {
				<-attestationIdleCh
				go node.AttestationIdle(cli, peernode, p, runtime, minerPoisInfo, teeRecord, l, cfg, attestationIdleCh)
			}

			if len(calcTagCh) > 0 {
				<-calcTagCh
				go node.CalcTag(cli, cace, l, runtime, teeRecord, cfg, wspace.GetFileDir(), calcTagCh)
			}

			if len(idleChallCh) > 0 || len(serviceChallCh) > 0 {
				go node.ChallengeMgt(cli, l, wspace, runtime, teeRecord, peernode, minerPoisInfo, rsakey, p, cfg, cace, idleChallCh, serviceChallCh)
				time.Sleep(chain.BlockInterval)
			}

			if len(genIdleCh) > 0 && !runtime.GetServiceChallengeFlag() && !runtime.GetIdleChallengeFlag() {
				<-genIdleCh
				go node.GenIdle(l, p.Prover, runtime, peernode.Workspace(), cfg.ReadUseSpace(), genIdleCh)
			}

		case <-tick_Hour.C:
			if runtime.GetMinerState() == chain.MINER_STATE_EXIT ||
				runtime.GetMinerState() == chain.MINER_STATE_OFFLINE {
				break
			}

			// go n.reportLogsMgt(ch_reportLogs)
			chainState = cli.GetRpcState()
			if !chainState {
				break
			}

			if len(replaceIdleCh) > 0 {
				<-replaceIdleCh
				go node.ReplaceIdle(cli, l, p, minerPoisInfo, teeRecord, peernode, replaceIdleCh)
			}

			if len(restoreCh) > 0 {
				<-restoreCh
				go node.RestoreFiles(cli, cace, l, wspace.GetFileDir(), restoreCh)
			}
		}
	}
}

func (n *Node) Reconnectrpc() {
	n.SetCurrentRpcst(false)
	if n.GetAndSetRpcConnecting() {
		return
	}
	defer n.SetRpcConnecting(false)

	n.Log("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	n.Ichal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	n.Schal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	out.Err(fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	err := n.ReconnectRpc()
	if err != nil {
		n.SetLastConnectedTime(time.Now().Format(time.DateTime))
		n.Log("err", "All RPCs failed to reconnect")
		n.Ichal("err", "All RPCs failed to reconnect")
		n.Schal("err", "All RPCs failed to reconnect")
		out.Err("All RPCs failed to reconnect")
		return
	}
	n.SetLastConnectedTime(time.Now().Format(time.DateTime))
	out.Ok(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.SetCurrentRpc(n.GetCurrentRpcAddr())
	n.SetCurrentRpcst(true)
}

func getCpuUsage(pid int32) float64 {
	p, _ := sprocess.NewProcess(pid)
	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0
	}
	return cpuPercent / float64(runtime.NumCPU())
}

func getMemUsage() uint64 {
	memSt := &runtime.MemStats{}
	runtime.ReadMemStats(memSt)
	return memSt.HeapSys + memSt.StackSys
}
