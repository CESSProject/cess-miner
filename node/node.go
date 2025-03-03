/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/node/logger"
	"github.com/CESSProject/cess-miner/node/record"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/confile"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	confile.Confiler
	logger.Logger
	cache.Cache
	record.TeeRecorder
	runstatus.Runstatus
	workspace.Workspace
	chain.Chainer
	*pb.MinerPoisInfo
	*RSAKeyPair
	*pois.Prover
	*acc.RsaKey
	*gin.Engine
	chain.ExpendersInfo
}

func NewEmptyNode() *Node {
	return &Node{}
}

func NewNodeWithConfig(cfg confile.Confiler) *Node {
	return &Node{Confiler: cfg}
}

func (n *Node) InitWorkspace(ws string) {
	n.Workspace = workspace.NewWorkspace(ws)
}

func (n *Node) InitChainclient(cli chain.Chainer) {
	n.Chainer = cli
}

func (n *Node) InitRSAKeyPair(key *RSAKeyPair) {
	n.RSAKeyPair = key
}

func (n *Node) InitTeeRecord(tees record.TeeRecorder) {
	n.TeeRecorder = tees
}

func (n *Node) InitMinerPoisInfo(poisInfo *pb.MinerPoisInfo) {
	n.MinerPoisInfo = poisInfo
}

func (n *Node) InitPoisProver(p *pois.Prover) {
	n.Prover = p
}

func (n *Node) InitAccRsaKey(key *acc.RsaKey) {
	n.RsaKey = key
}

func (n *Node) InitRunstatus(rt runstatus.Runstatus) {
	n.Runstatus = rt
}

func (n *Node) InitLogger(lg logger.Logger) {
	n.Logger = lg
}

func (n *Node) InitCacher(cace cache.Cache) {
	n.Cache = cace
}

func (n *Node) Start() {
	defer log.Println("Service has exited")
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	go exitHandle(exitCh)

	// check pois
	go n.CheckPois(int(n.ReadUseCpu()))

	// sync self info
	syncMinerStatusCh := make(chan bool, 1)
	go n.syncMinerStatus(syncMinerStatusCh)

	// sync tee info
	syncTeeCh := make(chan bool, 1)
	go n.SyncTeeInfo(syncTeeCh)

	reportFileCh := make(chan bool, 1)
	reportFileCh <- true

	genIdleCh := make(chan bool, 1)
	genIdleCh <- true

	certIdleCh := make(chan bool, 1)
	certIdleCh <- true

	calcTagCh := make(chan bool, 1)
	calcTagCh <- true

	idleChallCh := make(chan bool, 1)
	idleChallCh <- true

	serviceChallCh := make(chan bool, 1)
	serviceChallCh <- true

	replaceIdleCh := make(chan bool, 1)
	replaceIdleCh <- true

	restoreCh := make(chan bool, 1)
	restoreCh <- true

	tNow := time.Now().Unix()
	tOld_12s := tNow
	tOld_20s := tNow
	tOld_30s := tNow
	tOld_40s := tNow
	tOld_50s := tNow
	tOld_1m := tNow
	tOld_3m := tNow
	//tOld_10m := tNow
	tOld_1h := tNow

	out.Ok("Service started successfully")
	for {
		tNow = time.Now().Unix()

		// 10s challenge
		if tNow-tOld_12s >= 12 {
			if len(idleChallCh) > 0 || len(serviceChallCh) > 0 {
				go n.ChallengeMgt(idleChallCh, serviceChallCh, syncTeeCh)
				tOld_12s = tNow
			}
		}

		// 20s sync self info
		if tNow-tOld_20s >= 20 {
			if len(syncMinerStatusCh) > 0 {
				<-syncMinerStatusCh
				go n.syncMinerStatus(syncMinerStatusCh)
				tOld_20s = tNow
			}
		}

		// 30s report file
		if tNow-tOld_30s >= 30 {
			if len(reportFileCh) > 0 {
				<-reportFileCh
				go n.ReportFiles(reportFileCh)
				tOld_30s = tNow
			}
		}

		// 40s gen idle
		if tNow-tOld_40s >= 40 {
			if len(genIdleCh) > 0 {
				if !n.GetIdleChallenging() && !n.GetServiceChallenging() {
					<-genIdleCh
					go n.GenIdle(genIdleCh)
					tOld_40s = tNow
				}
			}
		}

		// 50s cert idle
		if tNow-tOld_50s >= 50 {
			if len(certIdleCh) > 0 {
				<-certIdleCh
				go n.CertIdle(certIdleCh)
				tOld_50s = tNow
			}
		}

		// 1m calc tag
		if tNow-tOld_1m >= 60 {
			if len(calcTagCh) > 0 {
				<-calcTagCh
				go n.CalcTag(calcTagCh)
				tOld_1m = tNow
			}
		}

		// 3m replace idle
		if tNow-tOld_3m >= 180 {
			if len(replaceIdleCh) > 0 {
				<-replaceIdleCh
				go n.ReplaceIdle(replaceIdleCh)
				tOld_3m = tNow
			}
		}

		// 10m sync tee info
		// if tNow-tOld_10m >= 600 {
		// 	if len(syncTeeCh) > 0 {
		// 		<-syncTeeCh
		// 		go n.SyncTeeInfo(syncTeeCh)
		// 		tOld_10m = tNow
		// 	}
		// }

		// 1h restore file
		if tNow-tOld_1h >= 3600 {
			if len(restoreCh) > 0 {
				<-restoreCh
				go n.RestoreFiles(restoreCh)
				tOld_1h = tNow
			}
		}
		time.Sleep(time.Millisecond * 100)
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
		// n.SetLastConnectedTime(time.Now().Format(time.DateTime))
		n.Log("err", "All RPCs failed to reconnect")
		n.Ichal("err", "All RPCs failed to reconnect")
		n.Schal("err", "All RPCs failed to reconnect")
		out.Err("All RPCs failed to reconnect")
		return
	}
	// n.SetLastConnectedTime(time.Now().Format(time.DateTime))
	out.Ok(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	n.SetCurrentRpc(n.GetCurrentRpcAddr())
	n.SetCurrentRpcst(true)
}

func (n *Node) CheckPois(cpus int) {
	n.SetCheckPois(true)
	defer n.SetCheckPois(false)

	cfg := pois.Config{
		AccPath:        n.GetPoisDir(),
		IdleFilePath:   n.GetSpaceDir(),
		ChallAccPath:   n.GetPoisAccDir(),
		MaxProofThread: int(n.ReadUseCpu()),
	}

	if n.GetRegister() {
		//Please initialize prover for the first time
		err := n.Prover.Init(*n.RsaKey, cfg)
		if err != nil {
			out.Err(fmt.Sprintf("pois prover init: %v", err))
			panic(fmt.Sprintf("pois prover init: %v", err))
		}
		n.Prover.AccManager.GetSnapshot()
		return
	}

	// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
	err := n.Prover.Recovery(*n.RsaKey, n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, cfg)
	if err != nil {
		if strings.Contains(err.Error(), "read element data") {
			num := 1
			// m, err := utils.GetSysMemAvailable()
			// cpuNum := runtime.NumCPU()
			// if err == nil {
			// 	m = m * 7 / 10 / (2 * 1024 * 1024 * 1024)
			// 	if int(m) < cpuNum {
			// 		cpuNum = int(m)
			// 	}
			// 	if cpuNum > num {
			// 		num = cpuNum
			// 	}
			// }
			if cpus > 1 {
				num = cpus
			}
			out.Tip(fmt.Sprintf("Check and restore idle data, used %d coroutines", num))
			err = n.Prover.CheckAndRestoreIdleData(n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, num)
			if err != nil {
				out.Err(fmt.Sprintf("check and restore idle data: %v", err))
				panic(fmt.Sprintf("check and restore idle data: %v", err))
			}
			err = n.Prover.Recovery(*n.RsaKey, n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, cfg)
			if err != nil {
				out.Err(fmt.Sprintf("pois prover recovery: %v", err))
				panic(fmt.Sprintf("pois prover recovery: %v", err))
			}
		} else {
			out.Err(fmt.Sprintf("pois prover recovery: %v", err))
			panic(fmt.Sprintf("pois prover recovery: %v", err))
		}
	}

	n.Prover.AccManager.GetSnapshot()
	out.Ok("Idle space check completed")
	return
}

func exitHandle(exitCh chan os.Signal) {
	for {
		select {
		case sig := <-exitCh:
			out.Tip(fmt.Sprintf("The program exits with the signal: %s", sig.String()))
			os.Exit(0)
		}
	}
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
