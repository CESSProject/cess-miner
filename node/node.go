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
	"github.com/CESSProject/cess-miner/pkg/utils"
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

	err := n.CheckPois()
	if err != nil {
		out.Err(fmt.Sprintf("check pois err: %v", err))
		os.Exit(1)
	}

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
				go n.ReportFiles(reportFileCh)
			}

			if len(attestationIdleCh) > 0 {
				<-attestationIdleCh
				go n.CertIdle(attestationIdleCh)
			}

			if len(calcTagCh) > 0 {
				<-calcTagCh
				go n.CalcTag(calcTagCh)
			}

			if len(idleChallCh) > 0 || len(serviceChallCh) > 0 {
				go n.ChallengeMgt(idleChallCh, serviceChallCh)
				time.Sleep(chain.BlockInterval)
			}

			if len(genIdleCh) > 0 && !n.GetIdleChallenging() && !n.GetServiceChallenging() {
				<-genIdleCh
				go n.GenIdle(genIdleCh)
			}

		case <-tick_Hour.C:
			if n.GetState() == chain.MINER_STATE_EXIT ||
				n.GetState() == chain.MINER_STATE_OFFLINE {
				break
			}

			chainState = n.GetRpcState()
			if !chainState {
				break
			}

			if len(replaceIdleCh) > 0 {
				<-replaceIdleCh
				go n.ReplaceIdle(replaceIdleCh)
			}

			if len(restoreCh) > 0 {
				<-restoreCh
				go n.RestoreFiles(restoreCh)
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

func (n *Node) CheckPois() error {
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
			return fmt.Errorf("pois prover init: %v", err)
		}
	} else {
		// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
		err := n.Prover.Recovery(*n.RsaKey, n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, cfg)
		if err != nil {
			if strings.Contains(err.Error(), "read element data") {
				num := 2
				m, err := utils.GetSysMemAvailable()
				cpuNum := runtime.NumCPU()
				if err == nil {
					m = m * 7 / 10 / (2 * 1024 * 1024 * 1024)
					if int(m) < cpuNum {
						cpuNum = int(m)
					}
					if cpuNum > num {
						num = cpuNum
					}
				}
				out.Tip(fmt.Sprintf("Check and restore idle data, use %d cpus", num))
				err = n.Prover.CheckAndRestoreIdleData(n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, num)
				if err != nil {
					return fmt.Errorf("check and restore idle data: %v", err)
				}
				err = n.Prover.Recovery(*n.RsaKey, n.MinerPoisInfo.Front, n.MinerPoisInfo.Rear, cfg)
				if err != nil {
					return fmt.Errorf("pois prover recovery: %v", err)
				}
			} else {
				return fmt.Errorf("pois prover recovery: %v", err)
			}
		}
	}
	n.Prover.AccManager.GetSnapshot()
	return nil
}

func exitHandle(exitCh chan os.Signal) {
	for {
		select {
		case sig := <-exitCh:
			panic(sig.String())
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
