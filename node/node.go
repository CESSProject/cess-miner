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
	"runtime"
	"strings"
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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	sdk.SDK
	core.P2P
	confile.Confile
	logger.Logger
	cache.Cache
	MinerState
	TeeRecord
	PeerRecord
	RunningRecord
	*gin.Engine
	*proof.RSAKeyPair
	*pb.MinerPoisInfo
	*DataDir
	*Pois
}

// New is used to build a empty node instance
func NewEmptyNode() *Node {
	return &Node{}
}

// New is used to build a node instance
func New() *Node {
	gin.SetMode(gin.ReleaseMode)
	return &Node{
		Engine:        gin.Default(),
		RSAKeyPair:    proof.NewKey(),
		TeeRecord:     NewTeeRecord(),
		MinerState:    NewMinerState(),
		PeerRecord:    NewPeerRecord(),
		RunningRecord: NewRunningRecord(),
		Pois:          &Pois{},
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
		ch_connectBoot      = make(chan bool, 1)
		ch_reportLogs       = make(chan bool, 1)
		ch_GenIdleFile      = make(chan bool, 1)
	)
	ch_calctag <- true
	ch_ConnectChain <- true
	ch_connectBoot <- true
	ch_idlechallenge <- true
	ch_servicechallenge <- true
	ch_reportfiles <- true
	ch_replace <- true
	ch_reportLogs <- true
	ch_GenIdleFile <- true
	ch_restoreMgt <- true

	// for {
	// 	out.Tip("QueryMasterPublicKey")
	// 	pubkey, err := n.QueryMasterPublicKey()
	// 	if err != nil {
	// 		out.Err(err.Error())
	// 		time.Sleep(pattern.BlockInterval)
	// 		continue
	// 	}
	// 	out.Err("SetPublickey")
	// 	err = n.SetPublickey(pubkey)
	// 	if err != nil {
	// 		time.Sleep(pattern.BlockInterval)
	// 		continue
	// 	}
	// 	n.Schal("info", "Initialize key successfully")
	// 	break
	// }

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
	//go n.findPeers(ch_findPeers)
	go n.recvPeers(ch_recvPeers)

	n.Log("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCores()))
	n.Log("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Ichal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCores()))
	n.Ichal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))
	n.Schal("info", fmt.Sprintf("Use %d cpu cores", n.GetCpuCores()))
	n.Schal("info", fmt.Sprintf("Use rpc: %s", n.GetCurrentRpcAddr()))

	out.Ok("Start successfully")

	for {
		select {
		case <-task_10S.C:
			n.SetTaskPeriod("10s")
			if len(ch_ConnectChain) > 0 {
				<-ch_ConnectChain
				go n.connectChain(ch_ConnectChain)
			}
			n.SetTaskPeriod("10s-end")

		case <-task_30S.C:
			n.SetTaskPeriod("30s")
			if len(ch_connectBoot) > 0 {
				<-ch_connectBoot
				go n.connectBoot(ch_connectBoot)
			}
			if len(ch_reportfiles) > 0 {
				<-ch_reportfiles
				go n.reportFiles(ch_reportfiles)
			}
			if len(ch_calctag) > 0 {
				<-ch_calctag
				go n.calcTag(ch_calctag)
			}
			n.SetTaskPeriod("30s-end")

		case <-task_Minute.C:
			n.SetTaskPeriod("1m")
			if len(ch_syncChainStatus) > 0 {
				<-ch_syncChainStatus
				go n.syncChainStatus(ch_syncChainStatus)
			}

			if len(ch_idlechallenge) > 0 || len(ch_servicechallenge) > 0 {
				go n.challengeMgt(ch_idlechallenge, ch_servicechallenge)
			}

			if len(ch_findPeers) > 0 {
				<-ch_findPeers
				//go n.findPeers(ch_findPeers)
			}

			if len(ch_recvPeers) > 0 {
				<-ch_recvPeers
				go n.recvPeers(ch_recvPeers)
			}

			if len(ch_GenIdleFile) > 0 {
				<-ch_GenIdleFile
				go n.genIdlefile(ch_GenIdleFile)
			}

			if len(ch_replace) > 0 {
				<-ch_replace
				go n.replaceIdle(ch_replace)
			}

			if len(ch_spaceMgt) > 0 {
				<-ch_spaceMgt
				go n.poisMgt(ch_spaceMgt)
			}

			if len(ch_restoreMgt) > 0 {
				<-ch_restoreMgt
				go n.restoreMgt(ch_restoreMgt)
			}
			n.SetTaskPeriod("1m-end")

		case <-task_Hour.C:
			n.SetTaskPeriod("1h")
			// go n.UpdatePeers()
			go n.reportLogsMgt(ch_reportLogs)
			n.SetTaskPeriod("1h-end")
		default:
			time.Sleep(time.Second)
		}
	}
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
	os.RemoveAll(n.DataDir.AccDir)
	os.RemoveAll(n.DataDir.PoisDir)
	os.RemoveAll(n.DataDir.RandomDir)
	os.MkdirAll(n.GetDirs().FileDir, pattern.DirMode)
	os.MkdirAll(n.GetDirs().TmpDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.DbDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.LogDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.SpaceDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.AccDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.PoisDir, pattern.DirMode)
	os.MkdirAll(n.DataDir.RandomDir, pattern.DirMode)
}

func (n *Node) ListenLocal() {
	var port uint32 = 6000
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET"}
	n.Engine.Use(cors.New(config))
	for {
		if !core.FreeLocalPort(port) {
			port++
		} else {
			break
		}
	}
	n.Engine.GET("/status", n.getStatusHandle)
	go n.Engine.Run(fmt.Sprintf(":%d", port))
	time.Sleep(time.Second)
	if !core.FreeLocalPort(port) {
		out.Tip(fmt.Sprintf("Listening on port: %d", port))
	}
}

// getStatusHandle
func (n *Node) getStatusHandle(c *gin.Context) {
	var msg string
	initStage := n.GetInitStage()
	if !strings.Contains(initStage[Stage_Complete], "[ok]") {
		msg += fmt.Sprintf("init stage: \n")
		for i := 0; i < len(initStage); i++ {
			msg += fmt.Sprintf("    %d: %s\n", i, initStage[i])
		}
	}
	msg += fmt.Sprintf("Process ID: %d\n", n.GetPID())

	msg += fmt.Sprintf("Task Stage: %s\n", n.GetTaskPeriod())

	msg += fmt.Sprintf("Miner State: %s\n", n.GetMinerState())

	if n.GetChainState() {
		msg += fmt.Sprintf("RPC Connection: [ok] %v\n", n.GetCurrentRpcAddr())
	} else {
		msg += fmt.Sprintf("RPC Connection: [fail] %v\n", n.GetCurrentRpcAddr())
	}
	msg += fmt.Sprintf("Last reconnection: %v\n", n.GetLastReconnectRpcTime())

	msg += fmt.Sprintf("Calculate Tag: %v\n", n.GetCalcTagFlag())

	msg += fmt.Sprintf("Report file: %v\n", n.GetReportFileFlag())

	msg += fmt.Sprintf("Generate idle: %v\n", n.GetGenIdleFlag())

	msg += fmt.Sprintf("Report idle: %v\n", n.GetAuthIdleFlag())

	msg += fmt.Sprintf("Cpu usage: %.2f%%\n", getCpuUsage(int32(n.GetPID())))

	msg += fmt.Sprintf("Memory usage: %d", getMemUsage())

	c.Data(200, "application/octet-stream", []byte(msg))
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
