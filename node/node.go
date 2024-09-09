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
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	confile.Confiler
	logger.Logger
	// cache.Cache
	TeeRecord
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

func (*Node) Start() {

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
