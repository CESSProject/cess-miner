/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"runtime"
	"sync"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/p2p-go/out"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	confile.Confiler
	logger.Logger
	cache.Cache
	TeeRecorder
	MinerRecord
	RunningStater
	*chain.ChainClient
	*pb.MinerPoisInfo
	*Workspace
	*RSAKeyPair
	*Pois
	*gin.Engine
	//*DataDir
	chain.ExpendersInfo
}

var (
	n    *Node
	once sync.Once
)

func GetNode() *Node {
	once.Do(func() {
		n = &Node{}
	})
	return n
}

func InitConfig(cfg confile.Confiler) {
	n := GetNode()
	n.Confiler = cfg
}

func InitWorkspace(path string) {
	n := GetNode()
	n.Workspace = &Workspace{
		rootDir: path,
	}
}

func InitChainclient(cli *chain.ChainClient) {
	n := GetNode()
	n.ChainClient = cli
}

func InitRSAKeyPair(key *RSAKeyPair) {
	n := GetNode()
	n.RSAKeyPair = key
}

func InitTeeRecord(tees *TeeRecord) {
	n := GetNode()
	n.TeeRecorder = tees
}

func InitMinerPoisInfo(poisInfo *pb.MinerPoisInfo) {
	n := GetNode()
	n.MinerPoisInfo = poisInfo
}

func InitPois(pois *Pois) {
	n := GetNode()
	n.Pois = pois
}

func InitLogger(lg logger.Logger) {
	n := GetNode()
	n.Logger = lg
}

func InitCacher(cace cache.Cache) {
	n := GetNode()
	n.Cache = cace
}

func (*Node) Start() {
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

	tick_29s := time.NewTicker(time.Second * time.Duration(29))
	defer tick_29s.Stop()

	tick_Minute := time.NewTicker(time.Second * time.Duration(57))
	defer tick_Minute.Stop()

	tick_Hour := time.NewTicker(time.Second * time.Duration(3597))
	defer tick_Hour.Stop()

	out.Ok("Service started successfully")
	// for {
	// 	select {
	// 	case <-tick_29s.C:
	// 		chainState = cli.GetRpcState()
	// 		if !chainState {
	// 			runtime.SetChainStatus(false)
	// 			runtime.SetReceiveFlag(false)
	// 			peernode.DisableRecv()
	// 			err = cli.ReconnectRpc()
	// 			l.Log("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	// 			l.Ichal("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	// 			l.Schal("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	// 			out.Err(fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
	// 			if err != nil {
	// 				runtime.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
	// 				l.Log("err", "All RPCs failed to reconnect")
	// 				l.Ichal("err", "All RPCs failed to reconnect")
	// 				l.Schal("err", "All RPCs failed to reconnect")
	// 				out.Err("All RPCs failed to reconnect")
	// 				break
	// 			}
	// 			runtime.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
	// 			out.Ok(fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
	// 			l.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
	// 			l.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
	// 			l.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
	// 			runtime.SetCurrentRpc(cli.GetCurrentRpcAddr())
	// 			runtime.SetChainStatus(true)
	// 			runtime.SetReceiveFlag(true)
	// 			peernode.EnableRecv()
	// 		}

	// 	case <-tick_Minute.C:
	// 		chainState = cli.GetRpcState()
	// 		if !chainState {
	// 			break
	// 		}

	// 		syncMinerStatus(cli, l, runtime)
	// 		if runtime.GetMinerState() == chain.MINER_STATE_EXIT ||
	// 			runtime.GetMinerState() == chain.MINER_STATE_OFFLINE {
	// 			break
	// 		}

	// 		if len(syncTeeCh) > 0 {
	// 			<-syncTeeCh
	// 			go node.SyncTeeInfo(cli, l, peernode, teeRecord, syncTeeCh)
	// 		}

	// 		if len(reportFileCh) > 0 {
	// 			<-reportFileCh
	// 			go node.ReportFiles(reportFileCh, cli, runtime, l, wspace.GetFileDir(), wspace.GetTmpDir())
	// 		}

	// 		if len(attestationIdleCh) > 0 {
	// 			<-attestationIdleCh
	// 			go node.AttestationIdle(cli, peernode, p, runtime, minerPoisInfo, teeRecord, l, cfg, attestationIdleCh)
	// 		}

	// 		if len(calcTagCh) > 0 {
	// 			<-calcTagCh
	// 			go node.CalcTag(cli, cace, l, runtime, teeRecord, cfg, wspace.GetFileDir(), calcTagCh)
	// 		}

	// 		if len(idleChallCh) > 0 || len(serviceChallCh) > 0 {
	// 			go node.ChallengeMgt(cli, l, wspace, runtime, teeRecord, peernode, minerPoisInfo, rsakey, p, cfg, cace, idleChallCh, serviceChallCh)
	// 			time.Sleep(chain.BlockInterval)
	// 		}

	// 		if len(genIdleCh) > 0 && !runtime.GetServiceChallengeFlag() && !runtime.GetIdleChallengeFlag() {
	// 			<-genIdleCh
	// 			go node.GenIdle(l, p.Prover, runtime, peernode.Workspace(), cfg.ReadUseSpace(), genIdleCh)
	// 		}

	// 	case <-tick_Hour.C:
	// 		if runtime.GetMinerState() == chain.MINER_STATE_EXIT ||
	// 			runtime.GetMinerState() == chain.MINER_STATE_OFFLINE {
	// 			break
	// 		}

	// 		// go n.reportLogsMgt(ch_reportLogs)
	// 		chainState = cli.GetRpcState()
	// 		if !chainState {
	// 			break
	// 		}

	// 		if len(replaceIdleCh) > 0 {
	// 			<-replaceIdleCh
	// 			go node.ReplaceIdle(cli, l, p, minerPoisInfo, teeRecord, peernode, replaceIdleCh)
	// 		}

	// 		if len(restoreCh) > 0 {
	// 			<-restoreCh
	// 			go node.RestoreFiles(cli, cace, l, wspace.GetFileDir(), restoreCh)
	// 		}
	// 	}
	// }
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
